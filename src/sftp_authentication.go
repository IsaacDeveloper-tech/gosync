package gosync

import (
	"errors"
	"fmt"
)

type SFTPAuthenticationSession interface {
	ServerHostKey() (RemoteSSHHostKey, error)
	Authenticate(username, password string) error
	Close() error
}

func authenticateSFTPSession(
	endpoint RemoteEndpoint,
	credentials RemoteCredentials,
	trustVerifier SSHHostTrustVerifier,
	session SFTPAuthenticationSession,
) error {
	if session == nil {
		return errors.New("SFTP authentication session is required")
	}
	if endpoint.Protocol != RemoteProtocolSFTP {
		return failSFTPAuthentication(session, errors.New("SFTP authentication requires an sftp destination"), credentials)
	}
	if err := validateRemoteEndpoint(endpoint); err != nil {
		return failSFTPAuthentication(session, err, credentials)
	}
	if !credentials.valid() {
		return failSFTPAuthentication(session, errors.New("remote credentials are unavailable"), credentials)
	}
	if trustVerifier == nil {
		return failSFTPAuthentication(session, errors.New("SFTP server identity verifier is unavailable"), credentials)
	}

	serverKey, err := session.ServerHostKey()
	if err != nil {
		return failSFTPAuthentication(session, fmt.Errorf("read SFTP server identity: %w", err), credentials)
	}
	if serverKey.Algorithm == "" || len(serverKey.Data) == 0 {
		return failSFTPAuthentication(session, errors.New("SFTP server identity is unavailable"), credentials)
	}
	trustStatus, trustErr := trustVerifier.CheckHostKey(endpoint.Host, endpoint.Port, serverKey)
	if trustErr != nil {
		trustFailureMessage := sftpHostTrustFailure(trustStatus)
		if trustStatus == SSHHostTrustRecognized {
			trustFailureMessage = errors.New("SFTP server identity verification failed")
		}
		trustFailure := fmt.Errorf("%w: %w", trustFailureMessage, trustErr)
		return failSFTPAuthentication(session, trustFailure, credentials)
	}
	if trustStatus != SSHHostTrustRecognized {
		return failSFTPAuthentication(session, sftpHostTrustFailure(trustStatus), credentials)
	}
	if err := session.Authenticate(credentials.Username(), credentials.Password()); err != nil {
		return failSFTPAuthentication(session, fmt.Errorf("SFTP authentication failed: %w", err), credentials)
	}
	return nil
}

func sftpHostTrustFailure(status SSHHostTrustStatus) error {
	switch status {
	case SSHHostTrustAbsent:
		return errors.New("SFTP server is not trusted for this host and port")
	case SSHHostTrustChanged:
		return errors.New("SFTP server identity has changed")
	case SSHHostTrustUnavailable:
		return errors.New("SFTP server identity could not be verified")
	default:
		return errors.New("SFTP server identity is not trusted for this host and port")
	}
}

func failSFTPAuthentication(session SFTPAuthenticationSession, primaryError error, credentials RemoteCredentials) error {
	var closeError error
	if session != nil {
		closeError = session.Close()
	}
	return sanitizeRemoteError(combineRemoteOperationErrors(primaryError, nil, closeError), credentials)
}
