package gosync

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const maximumRemoteCredentialLength = 4096

type RemoteCredentials struct {
	username string
	password string
}

type RemoteWatchExecution struct {
	Policy      RemoteWatchPolicySnapshot
	Credentials RemoteCredentials
	Logger      *LoggingCoordinator
}

type protectedRemoteError struct {
	message string
	cause   error
}

func newRemoteCredentials(username, password string) (RemoteCredentials, error) {
	if !validRemoteCredential(username) {
		return RemoteCredentials{}, errors.New("remote username must not be empty or contain line breaks")
	}
	if !validRemoteCredential(password) {
		return RemoteCredentials{}, errors.New("remote password must not be empty or contain line breaks")
	}
	return RemoteCredentials{username: username, password: password}, nil
}

func (credentials RemoteCredentials) Username() string {
	return credentials.username
}

func (credentials RemoteCredentials) Password() string {
	return credentials.password
}

func (credentials RemoteCredentials) String() string {
	return "[REDACTED REMOTE CREDENTIALS]"
}

func (credentials RemoteCredentials) GoString() string {
	return credentials.String()
}

func (execution RemoteWatchExecution) String() string {
	if endpointURL := execution.Policy.Destination.SanitizedURL(); endpointURL != "" {
		return "remote watch execution for " + endpointURL + " [credentials redacted]"
	}
	return "remote watch execution [credentials redacted]"
}

func (execution RemoteWatchExecution) GoString() string {
	return execution.String()
}

func (credentials RemoteCredentials) valid() bool {
	return validRemoteCredential(credentials.username) && validRemoteCredential(credentials.password)
}

func validRemoteCredential(value string) bool {
	return value != "" && strings.TrimSpace(value) != "" && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}

func promptRemoteCredentials(input io.Reader, output io.Writer) (RemoteCredentials, error) {
	if input == nil {
		return RemoteCredentials{}, errors.New("remote credential input is unavailable")
	}
	if output == nil {
		return RemoteCredentials{}, errors.New("remote credential prompt output is unavailable")
	}

	username, err := promptRemoteCredentialLine(input, output, "Remote username: ", "read remote username")
	if err != nil {
		return RemoteCredentials{}, err
	}
	password, err := promptRemoteCredentialLine(input, output, "Remote password: ", "read remote password", username)
	if err != nil {
		return RemoteCredentials{}, err
	}
	credentials, err := newRemoteCredentials(username, password)
	if err != nil {
		return RemoteCredentials{}, err
	}
	return credentials, nil
}

func promptRemoteCredentialLine(input io.Reader, output io.Writer, prompt, operation string, protectedValues ...string) (string, error) {
	writtenCount, err := io.WriteString(output, prompt)
	if err != nil {
		return "", protectCredentialError(operation+" prompt failed", err, protectedValues...)
	}
	if writtenCount != len(prompt) {
		return "", protectCredentialError(operation+" prompt failed", io.ErrShortWrite, protectedValues...)
	}
	credential, readErr := readRemoteCredentialLine(input)
	if readErr != nil {
		knownValues := append(append([]string(nil), protectedValues...), credential)
		return "", protectCredentialError(operation+" failed", readErr, knownValues...)
	}
	if !validRemoteCredential(credential) {
		return "", fmt.Errorf("%s is empty or contains unsupported line breaks", strings.TrimPrefix(operation, "read "))
	}
	return credential, nil
}

func readRemoteCredentialLine(input io.Reader) (string, error) {
	var credential strings.Builder
	var character [1]byte
	noProgressReads := 0
	for {
		readCount, err := input.Read(character[:])
		if readCount > 0 {
			noProgressReads = 0
			if character[0] == '\n' {
				value := strings.TrimSuffix(credential.String(), "\r")
				return value, nil
			}
			if credential.Len() >= maximumRemoteCredentialLength {
				return credential.String(), errors.New("credential input exceeds the supported length")
			}
			credential.WriteByte(character[0])
		}
		if err != nil {
			if errors.Is(err, io.EOF) && credential.Len() > 0 {
				return credential.String(), io.ErrUnexpectedEOF
			}
			return credential.String(), err
		}
		if readCount == 0 {
			noProgressReads++
			if noProgressReads >= 100 {
				return credential.String(), io.ErrNoProgress
			}
		}
	}
}

func protectCredentialError(message string, cause error, protectedValues ...string) error {
	if cause == nil {
		return errors.New(message)
	}
	sanitizer := newLogSanitizer()
	for _, protectedValue := range protectedValues {
		sanitizer.RegisterProtectedValue(protectedValue)
	}
	return &protectedRemoteError{message: sanitizer.SanitizeMessage(message + ": " + cause.Error()), cause: cause}
}

func sanitizeRemoteError(remoteError error, credentials RemoteCredentials) error {
	if remoteError == nil {
		return nil
	}
	sanitizer := newLogSanitizer()
	sanitizer.RegisterProtectedValue(credentials.username)
	sanitizer.RegisterProtectedValue(credentials.password)
	sanitizedMessage := sanitizer.SanitizeMessage(remoteError.Error())
	if sanitizedMessage == remoteError.Error() {
		return remoteError
	}
	return &protectedRemoteError{message: sanitizedMessage, cause: remoteError}
}

func (remoteError *protectedRemoteError) Error() string {
	if remoteError == nil {
		return ""
	}
	return remoteError.message
}

func (remoteError *protectedRemoteError) Unwrap() error {
	if remoteError == nil {
		return nil
	}
	return remoteError.cause
}
