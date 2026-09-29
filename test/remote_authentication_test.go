package gosync_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRemoteCredentialsRemainSeparateAndRedacted(t *testing.T) {
	credentials, err := newRemoteCredentials("alice-private", "password-private")
	if err != nil {
		t.Fatalf("NewRemoteCredentials() error = %v", err)
	}
	if credentials.Username() != "alice-private" || credentials.Password() != "password-private" {
		t.Fatal("remote credentials did not retain the supplied values")
	}

	for name, rendered := range map[string]string{
		"string":    fmt.Sprint(credentials),
		"go string": fmt.Sprintf("%#v", credentials),
		"format":    fmt.Sprintf("%+v", credentials),
	} {
		if strings.Contains(rendered, "alice-private") || strings.Contains(rendered, "password-private") {
			t.Errorf("%s formatting exposed credentials: %q", name, rendered)
		}
	}
	encodedCredentials, err := json.Marshal(credentials)
	if err != nil {
		t.Fatalf("Marshal(credentials) error = %v", err)
	}
	if strings.Contains(string(encodedCredentials), "alice-private") || strings.Contains(string(encodedCredentials), "password-private") {
		t.Fatalf("serialized credentials exposed a protected value: %s", encodedCredentials)
	}
	for index := 0; index < reflect.TypeOf(credentials).NumField(); index++ {
		if reflect.TypeOf(credentials).Field(index).IsExported() {
			t.Fatalf("credential field %q is exported", reflect.TypeOf(credentials).Field(index).Name)
		}
	}
	for _, domainModel := range []any{RemoteEndpoint{}, RemoteWatchPolicySnapshot{}, ConfirmedSynchronizationState{}, BackupSetState{}, BackupArchiveIdentity{}} {
		modelType := reflect.TypeOf(domainModel)
		for index := 0; index < modelType.NumField(); index++ {
			fieldName := strings.ToLower(modelType.Field(index).Name)
			if strings.Contains(fieldName, "credential") || strings.Contains(fieldName, "password") || strings.Contains(fieldName, "username") {
				t.Errorf("%s contains protected field %q", modelType.Name(), modelType.Field(index).Name)
			}
		}
	}

	request, err := parseWatchCommandRequest([]string{"watch", t.TempDir(), "sftp://server.example/archive"})
	if err != nil {
		t.Fatalf("ParseWatchCommandRequest() error = %v", err)
	}
	configuration := ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 5, SynchronizationMode: SynchronizationModeUnidirectional}
	policy, err := buildRemoteWatchPolicySnapshot(request, configuration)
	if err != nil {
		t.Fatalf("BuildRemoteWatchPolicySnapshot() error = %v", err)
	}
	if strings.Contains(policy.Destination.SanitizedURL(), "alice-private") || strings.Contains(policy.Destination.SanitizedURL(), "password-private") {
		t.Fatalf("endpoint display included credentials: %q", policy.Destination.SanitizedURL())
	}
	encodedPolicy, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("Marshal(remote policy) error = %v", err)
	}
	if strings.Contains(string(encodedPolicy), "alice-private") || strings.Contains(string(encodedPolicy), "password-private") {
		t.Fatalf("remote policy serialization exposed credentials: %s", encodedPolicy)
	}
	execution := RemoteWatchExecution{Policy: policy, Credentials: credentials}
	if formattedExecution := fmt.Sprintf("%+v", execution); strings.Contains(formattedExecution, "alice-private") || strings.Contains(formattedExecution, "password-private") {
		t.Fatalf("remote execution formatting exposed credentials: %q", formattedExecution)
	}
}

func TestPromptRemoteCredentialsReadsBothValuesOnceAndPreservesPasswordSpaces(t *testing.T) {
	var output bytes.Buffer
	credentials, err := promptRemoteCredentials(strings.NewReader("alice\npass phrase with spaces\n"), &output)
	if err != nil {
		t.Fatalf("PromptRemoteCredentials() error = %v", err)
	}
	if credentials.Username() != "alice" || credentials.Password() != "pass phrase with spaces" {
		t.Fatalf("credentials = %s, want exact username and password input", credentials)
	}
	if output.String() != "Remote username: Remote password: " {
		t.Fatalf("credential prompts = %q, want one prompt for each value", output.String())
	}
}

func TestPromptRemoteCredentialsRejectsUnavailableCancelledAndIncompleteInput(t *testing.T) {
	if _, err := promptRemoteCredentials(strings.NewReader("alice\npass\n"), nil); err == nil {
		t.Fatal("PromptRemoteCredentials(nil output) error = nil, want unavailable-output failure")
	}
	testCases := []struct {
		name  string
		input io.Reader
		want  error
	}{
		{name: "missing reader", input: nil},
		{name: "username EOF", input: strings.NewReader(""), want: io.EOF},
		{name: "password EOF", input: strings.NewReader("alice\n"), want: io.EOF},
		{name: "cancelled", input: failingReader{err: context.Canceled}, want: context.Canceled},
		{name: "read failure after username", input: &stagedReader{firstLine: "alice\n", err: errCredentialInputFailure}, want: errCredentialInputFailure},
		{name: "unterminated username", input: strings.NewReader("alice"), want: io.ErrUnexpectedEOF},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var output bytes.Buffer
			credentials, err := promptRemoteCredentials(testCase.input, &output)
			if err == nil {
				t.Fatalf("PromptRemoteCredentials() = %s, error = nil, want input failure", credentials)
			}
			if testCase.want != nil && !errors.Is(err, testCase.want) {
				t.Fatalf("PromptRemoteCredentials() error = %v, want errors.Is(_, %v)", err, testCase.want)
			}
			if strings.Contains(err.Error(), "alice") {
				t.Fatalf("credential input error disclosed username: %v", err)
			}
		})
	}
}

func TestPromptRemoteCredentialsRejectsEmptyValuesAndPropagatesPromptFailures(t *testing.T) {
	for _, input := range []string{"\npassword\n", "alice\n   \n"} {
		if _, err := promptRemoteCredentials(strings.NewReader(input), io.Discard); err == nil {
			t.Fatalf("PromptRemoteCredentials(%q) error = nil, want empty credential rejection", input)
		}
	}

	for _, failingWrite := range []int{1, 2} {
		t.Run(fmt.Sprintf("writer failure %d", failingWrite), func(t *testing.T) {
			reader := &countingReader{reader: strings.NewReader("alice\npassword\n")}
			writer := &credentialFailingWriter{failOnWrite: failingWrite, err: errCredentialOutputFailure}
			_, err := promptRemoteCredentials(reader, writer)
			if !errors.Is(err, errCredentialOutputFailure) {
				t.Fatalf("PromptRemoteCredentials() error = %v, want prompt-output cause", err)
			}
			if writer.writeCount != failingWrite {
				t.Fatalf("prompt writes = %d, want %d", writer.writeCount, failingWrite)
			}
			if failingWrite == 1 && reader.readCount != 0 {
				t.Fatalf("input reads = %d, want no read before username prompt succeeds", reader.readCount)
			}
			if failingWrite == 2 && reader.readCount != len("alice\n") {
				t.Fatalf("input bytes read = %d, want username only before password prompt succeeds", reader.readCount)
			}
		})
	}
	if _, err := promptRemoteCredentials(strings.NewReader("alice\npass\n"), shortCredentialWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("PromptRemoteCredentials(short output) error = %v, want io.ErrShortWrite", err)
	}
}

func TestRemoteWatchPromptsOnceAndProvidesCredentialsOutsideEndpoint(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "data.bin"), []byte{0, 0xff, 1}, 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	stop := make(chan struct{})
	var output bytes.Buffer
	var received []RemoteWatchExecution
	watchCalls := 0
	err := runWatchCommand([]string{"watch", sourceRoot, "ftp://server.example/archive"}, WatchCommandOptions{
		Input:              strings.NewReader("alice\npass phrase\n"),
		Output:             &output,
		Stop:               stop,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logging:            LoggingCoordinatorOptions{FileDirectory: filepath.Join(temporaryDirectory, "logs")},
		Wait:               func(time.Duration) {},
		SynchronizeRemoteWithPolicy: func(execution RemoteWatchExecution) error {
			received = append(received, execution)
			watchCalls++
			if watchCalls == 2 {
				close(stop)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWatchCommand() error = %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("remote execution calls = %d, want two cycles", len(received))
	}
	for _, execution := range received {
		if execution.Credentials.Username() != "alice" || execution.Credentials.Password() != "pass phrase" {
			t.Fatalf("execution credentials = %s, want one execution-scoped pair", execution.Credentials)
		}
		if strings.Contains(execution.Policy.Destination.SanitizedURL(), "alice") || strings.Contains(execution.Policy.Destination.SanitizedURL(), "pass") {
			t.Fatalf("execution endpoint contains credentials: %q", execution.Policy.Destination.SanitizedURL())
		}
	}
	if output.String() != "Remote username: Remote password: " {
		t.Fatalf("prompt output = %q, want one username/password prompt for the entire watch", output.String())
	}
	persistedConfiguration, err := os.ReadFile(configurationStore.Path())
	if err != nil {
		t.Fatalf("ReadFile(configuration) error = %v", err)
	}
	if strings.Contains(string(persistedConfiguration), "alice") || strings.Contains(string(persistedConfiguration), "pass phrase") {
		t.Fatalf("persisted configuration contains credentials: %s", persistedConfiguration)
	}
}

func TestRemoteWatchDoesNotDispatchWhenCredentialInputOrPromptFails(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	logger, err := newLoggingCoordinator(LoggingCoordinatorOptions{ConsoleWriter: io.Discard})
	if err != nil {
		t.Fatalf("NewLoggingCoordinator() error = %v", err)
	}
	testCases := []struct {
		name   string
		input  io.Reader
		output io.Writer
	}{
		{name: "username EOF", input: strings.NewReader(""), output: io.Discard},
		{name: "password EOF", input: strings.NewReader("alice\n"), output: io.Discard},
		{name: "input cancellation", input: failingReader{err: context.Canceled}, output: io.Discard},
		{name: "prompt output failure", input: strings.NewReader("alice\npassword\n"), output: &credentialFailingWriter{failOnWrite: 1, err: errCredentialOutputFailure}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dispatchCalls := 0
			err := runWatchCommand([]string{"watch", sourceRoot, "sftp://server.example/archive"}, WatchCommandOptions{
				Input:              testCase.input,
				Output:             testCase.output,
				StateStore:         &stateStore,
				ConfigurationStore: &configurationStore,
				Logger:             logger,
				SynchronizeRemoteWithPolicy: func(RemoteWatchExecution) error {
					dispatchCalls++
					return nil
				},
			})
			if err == nil {
				t.Fatal("runWatchCommand() error = nil, want credential failure")
			}
			if dispatchCalls != 0 {
				t.Fatalf("remote dispatch calls = %d, want 0 before credentials are available", dispatchCalls)
			}
		})
	}
}

func TestRemoteWatchRedactsCredentialsFromLogsAndReturnedServerErrors(t *testing.T) {
	temporaryDirectory := t.TempDir()
	sourceRoot := filepath.Join(temporaryDirectory, "source")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	configurationStore := newConfigurationStore(filepath.Join(temporaryDirectory, "config.json"))
	configuration := ConfigurationSnapshot{SchemaVersion: 2, SynchronizationIntervalSeconds: 1, SynchronizationMode: SynchronizationModeUnidirectional}
	if err := configurationStore.Save(configuration); err != nil {
		t.Fatalf("Save(configuration) error = %v", err)
	}
	stateStore := newConfirmedStateStoreAt(filepath.Join(temporaryDirectory, "state"))
	var logOutput bytes.Buffer
	logger, err := newLoggingCoordinator(LoggingCoordinatorOptions{ConsoleWriter: &logOutput})
	if err != nil {
		t.Fatalf("NewLoggingCoordinator() error = %v", err)
	}
	serverError := errors.New("server echoed login-private and password-private")
	err = runWatchCommand([]string{"watch", sourceRoot, "sftp://server.example/archive"}, WatchCommandOptions{
		Input:              strings.NewReader("login-private\npassword-private\n"),
		Output:             io.Discard,
		StateStore:         &stateStore,
		ConfigurationStore: &configurationStore,
		Logger:             logger,
		SynchronizeRemoteWithPolicy: func(execution RemoteWatchExecution) error {
			if err := execution.Logger.Log(LogEntry{
				Severity: LogSeverityError,
				Event:    LogEventOperationFailed,
				Message:  serverError.Error(),
				Context:  map[string]string{"remote_error": serverError.Error()},
			}); err != nil {
				return err
			}
			return serverError
		},
	})
	if !errors.Is(err, serverError) {
		t.Fatalf("runWatchCommand() error = %v, want wrapped server cause", err)
	}
	for _, protectedValue := range []string{"login-private", "password-private"} {
		if strings.Contains(err.Error(), protectedValue) {
			t.Errorf("returned remote error disclosed %q: %v", protectedValue, err)
		}
		if strings.Contains(logOutput.String(), protectedValue) {
			t.Errorf("remote log disclosed %q: %s", protectedValue, logOutput.String())
		}
	}
	if !strings.Contains(logOutput.String(), "[REDACTED]") {
		t.Fatalf("remote log = %q, want redaction marker while retaining the failure message", logOutput.String())
	}
}

func TestSSHHostTrustStoreChecksKnownHostsByHostAndPort(t *testing.T) {
	temporaryDirectory := t.TempDir()
	knownHostsPath := filepath.Join(temporaryDirectory, "known_hosts")
	trustedKey := RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{1, 2, 3, 4, 5}}
	changedKey := RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{9, 8, 7, 6, 5}}
	writeKnownHosts(t, knownHostsPath,
		knownHostsLine("[git.example]:2222", trustedKey)+
			knownHostsLine(hashedKnownHost("hashed.example", []byte("salt-value")), trustedKey)+
			knownHostsLine("other.example", trustedKey),
	)
	trustStore := newSSHHostTrustStoreAt(knownHostsPath)

	testCases := []struct {
		name    string
		host    string
		port    int
		key     RemoteSSHHostKey
		want    SSHHostTrustStatus
		wantErr bool
	}{
		{name: "matching host key", host: "git.example", port: 2222, key: trustedKey, want: SSHHostTrustRecognized},
		{name: "hashed known host", host: "hashed.example", port: 22, key: trustedKey, want: SSHHostTrustRecognized},
		{name: "unknown host", host: "new.example", port: 2222, key: trustedKey, want: SSHHostTrustAbsent},
		{name: "changed key", host: "git.example", port: 2222, key: changedKey, want: SSHHostTrustChanged},
		{name: "different port", host: "git.example", port: 22, key: trustedKey, want: SSHHostTrustAbsent},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := trustStore.CheckHostKey(testCase.host, testCase.port, testCase.key)
			if status != testCase.want || (err != nil) != testCase.wantErr {
				t.Fatalf("CheckHostKey(%q, %d) = (%q, %v), want (%q, err=%t)", testCase.host, testCase.port, status, err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestSSHHostTrustStoreDistinguishesAbsentFromUnavailable(t *testing.T) {
	testCases := []struct {
		name  string
		store SSHHostTrustStore
		want  SSHHostTrustStatus
		err   bool
	}{
		{name: "known_hosts file absent", store: newSSHHostTrustStoreAt(filepath.Join(t.TempDir(), "missing_known_hosts")), want: SSHHostTrustAbsent},
		{name: "trust source unavailable", store: newSSHHostTrustStoreAt(""), want: SSHHostTrustUnavailable, err: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status, err := testCase.store.CheckHostKey("git.example", 22, RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{1}})
			if status != testCase.want || (err != nil) != testCase.err {
				t.Fatalf("CheckHostKey() = (%q, %v), want (%q, err=%t)", status, err, testCase.want, testCase.err)
			}
		})
	}
}

func TestSFTPAuthenticationRefusesCredentialsBeforeTrustedHostKey(t *testing.T) {
	credentials, err := newRemoteCredentials("login-private", "password-private")
	if err != nil {
		t.Fatalf("NewRemoteCredentials() error = %v", err)
	}
	endpoint := RemoteEndpoint{Protocol: RemoteProtocolSFTP, Host: "git.example", Port: 2222, Path: "/backups"}
	testCases := []struct {
		name     string
		status   SSHHostTrustStatus
		trustErr error
		want     string
	}{
		{name: "absent trust", status: SSHHostTrustAbsent, want: "not trusted"},
		{name: "changed trust", status: SSHHostTrustChanged, want: "changed"},
		{name: "unavailable trust", status: SSHHostTrustUnavailable, trustErr: errors.New("server echoed password-private"), want: "could not be verified"},
		{name: "unknown trust outcome", status: SSHHostTrustStatus("unexpected"), want: "not trusted"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			verifier := &fixedSSHHostTrustVerifier{status: testCase.status, err: testCase.trustErr}
			session := &fakeSFTPAuthenticationSession{hostKey: RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{3, 4, 5}}}
			err := authenticateSFTPSession(endpoint, credentials, verifier, session)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.want)) {
				t.Fatalf("AuthenticateSFTPSession() error = %v, want %q", err, testCase.want)
			}
			if session.authenticateCalls != 0 {
				t.Fatalf("password authentication attempts = %d, want 0 before host trust", session.authenticateCalls)
			}
			if session.closeCalls != 1 {
				t.Fatalf("session close calls = %d, want one cleanup after refusing trust", session.closeCalls)
			}
			if verifier.host != "git.example" || verifier.port != 2222 {
				t.Fatalf("trust lookup used host %q port %d, want host-and-port identity", verifier.host, verifier.port)
			}
			if strings.Contains(err.Error(), "login-private") || strings.Contains(err.Error(), "password-private") {
				t.Fatalf("trust error disclosed credentials: %v", err)
			}
		})
	}

	invalidKeySession := &fakeSFTPAuthenticationSession{}
	err = authenticateSFTPSession(endpoint, credentials, &fixedSSHHostTrustVerifier{status: SSHHostTrustRecognized}, invalidKeySession)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "identity is unavailable") {
		t.Fatalf("AuthenticateSFTPSession(empty server key) error = %v, want unverifiable identity", err)
	}
	if invalidKeySession.authenticateCalls != 0 {
		t.Fatalf("empty-key authentication attempts = %d, want 0", invalidKeySession.authenticateCalls)
	}
}

func TestSFTPAuthenticationSendsCredentialsOnlyAfterTrustAndRedactsAuthErrors(t *testing.T) {
	credentials, err := newRemoteCredentials("login-private", "password-private")
	if err != nil {
		t.Fatalf("NewRemoteCredentials() error = %v", err)
	}
	endpoint := RemoteEndpoint{Protocol: RemoteProtocolSFTP, Host: "git.example", Port: 22, Path: "/backups"}
	trustedVerifier := &fixedSSHHostTrustVerifier{status: SSHHostTrustRecognized}
	trustedSession := &fakeSFTPAuthenticationSession{hostKey: RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{1, 2, 3}}}
	if err := authenticateSFTPSession(endpoint, credentials, trustedVerifier, trustedSession); err != nil {
		t.Fatalf("AuthenticateSFTPSession(trusted) error = %v", err)
	}
	if trustedVerifier.host != "git.example" || trustedVerifier.port != 22 || trustedSession.authenticateCalls != 1 {
		t.Fatalf("trusted auth sequence = verifier(%q,%d), authenticate calls=%d", trustedVerifier.host, trustedVerifier.port, trustedSession.authenticateCalls)
	}
	if trustedSession.username != "login-private" || trustedSession.password != "password-private" || trustedSession.closeCalls != 0 {
		t.Fatalf("trusted session = username %q, password %q, close calls %d", trustedSession.username, trustedSession.password, trustedSession.closeCalls)
	}

	authenticationError := errors.New("server rejected login-private with password-private")
	failedSession := &fakeSFTPAuthenticationSession{
		hostKey: RemoteSSHHostKey{Algorithm: "ssh-ed25519", Data: []byte{1, 2, 3}},
		authErr: authenticationError,
	}
	err = authenticateSFTPSession(endpoint, credentials, trustedVerifier, failedSession)
	if !errors.Is(err, authenticationError) {
		t.Fatalf("authentication error = %v, want original cause", err)
	}
	if strings.Contains(err.Error(), "login-private") || strings.Contains(err.Error(), "password-private") {
		t.Fatalf("authentication error disclosed credentials: %v", err)
	}
	if failedSession.closeCalls != 1 {
		t.Fatalf("failed session close calls = %d, want one", failedSession.closeCalls)
	}
}

func TestSFTPAuthenticationPreservesHostKeyAndCloseFailures(t *testing.T) {
	credentials, err := newRemoteCredentials("user", "pass")
	if err != nil {
		t.Fatalf("NewRemoteCredentials() error = %v", err)
	}
	primaryError := errors.New("host key exchange failed")
	closeError := errors.New("session close failed")
	session := &fakeSFTPAuthenticationSession{hostKeyErr: primaryError, closeErr: closeError}
	err = authenticateSFTPSession(
		RemoteEndpoint{Protocol: RemoteProtocolSFTP, Host: "git.example", Port: 22, Path: "/backups"},
		credentials,
		&fixedSSHHostTrustVerifier{status: SSHHostTrustRecognized},
		session,
	)
	if !errors.Is(err, primaryError) || !errors.Is(err, closeError) {
		t.Fatalf("authentication failure = %v, want both host-key and close errors", err)
	}
	var failure *RemoteOperationFailure
	if !errors.As(err, &failure) || failure.OperationError == nil || failure.CloseError == nil {
		t.Fatalf("authentication failure type = %T (%v), want distinct operation and close causes", err, err)
	}
	if session.authenticateCalls != 0 {
		t.Fatalf("authentication attempts = %d, want 0 after host-key exchange failure", session.authenticateCalls)
	}
}

func hashedKnownHost(host string, salt []byte) string {
	mac := hmac.New(sha1.New, salt)
	_, _ = mac.Write([]byte(host))
	return "|1|" + base64.StdEncoding.EncodeToString(salt) + "|" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type fixedSSHHostTrustVerifier struct {
	status SSHHostTrustStatus
	err    error
	host   string
	port   int
}

func (verifier *fixedSSHHostTrustVerifier) CheckHostKey(host string, port int, _ RemoteSSHHostKey) (SSHHostTrustStatus, error) {
	verifier.host = host
	verifier.port = port
	return verifier.status, verifier.err
}

type fakeSFTPAuthenticationSession struct {
	hostKey           RemoteSSHHostKey
	hostKeyErr        error
	authErr           error
	closeErr          error
	username          string
	password          string
	hostKeyCalls      int
	authenticateCalls int
	closeCalls        int
}

func (session *fakeSFTPAuthenticationSession) ServerHostKey() (RemoteSSHHostKey, error) {
	session.hostKeyCalls++
	return session.hostKey, session.hostKeyErr
}

func (session *fakeSFTPAuthenticationSession) Authenticate(username, password string) error {
	session.authenticateCalls++
	session.username = username
	session.password = password
	return session.authErr
}

func (session *fakeSFTPAuthenticationSession) Close() error {
	session.closeCalls++
	return session.closeErr
}

var (
	errCredentialInputFailure  = errors.New("credential input failed")
	errCredentialOutputFailure = errors.New("credential prompt output failed")
)

type failingReader struct {
	err error
}

func (reader failingReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type stagedReader struct {
	firstLine string
	err       error
}

func (reader *stagedReader) Read(buffer []byte) (int, error) {
	if len(reader.firstLine) == 0 {
		return 0, reader.err
	}
	readCount := copy(buffer, reader.firstLine)
	reader.firstLine = reader.firstLine[readCount:]
	return readCount, nil
}

type countingReader struct {
	reader    io.Reader
	readCount int
}

func (reader *countingReader) Read(buffer []byte) (int, error) {
	reader.readCount++
	if len(buffer) > 1 {
		buffer = buffer[:1]
	}
	return reader.reader.Read(buffer)
}

type credentialFailingWriter struct {
	failOnWrite int
	err         error
	writeCount  int
}

type shortCredentialWriter struct{}

func (shortCredentialWriter) Write(buffer []byte) (int, error) {
	return len(buffer) - 1, nil
}

func (writer *credentialFailingWriter) Write(buffer []byte) (int, error) {
	writer.writeCount++
	if writer.writeCount == writer.failOnWrite {
		return 0, writer.err
	}
	return len(buffer), nil
}

func knownHostsLine(host string, key RemoteSSHHostKey) string {
	return host + " " + key.Algorithm + " " + base64.StdEncoding.EncodeToString(key.Data) + " test fixture\n"
}

func writeKnownHosts(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(known_hosts) error = %v", err)
	}
}
