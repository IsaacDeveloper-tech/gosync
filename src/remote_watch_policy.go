package gosync

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type RemoteWatchPolicySnapshot struct {
	SourceRoot           string
	Destination          RemoteEndpoint
	Mode                 SynchronizationMode
	Interval             time.Duration
	BackupRetentionCount int
}

func buildRemoteWatchPolicySnapshot(request WatchCommandRequest, configuration ConfigurationSnapshot) (RemoteWatchPolicySnapshot, error) {
	if request.Destination.Kind != WatchDestinationRemote {
		return RemoteWatchPolicySnapshot{}, fmt.Errorf("remote watch policy requires a remote destination")
	}
	if err := validateRemoteEndpoint(request.Destination.RemoteEndpoint); err != nil {
		return RemoteWatchPolicySnapshot{}, err
	}
	if err := validateConfigurationSnapshot(configuration); err != nil {
		return RemoteWatchPolicySnapshot{}, fmt.Errorf("validate remote watch configuration: %w", err)
	}
	if configuration.SynchronizationMode != SynchronizationModeUnidirectional && configuration.SynchronizationMode != SynchronizationModeBackup {
		return RemoteWatchPolicySnapshot{}, fmt.Errorf("remote destinations support only unidirectional and BACKUP modes")
	}

	sourceRoot, err := normalizeRootPath(request.SourceRoot)
	if err != nil {
		return RemoteWatchPolicySnapshot{}, fmt.Errorf("normalize remote watch source directory: %w", err)
	}
	return RemoteWatchPolicySnapshot{
		SourceRoot:           sourceRoot,
		Destination:          request.Destination.RemoteEndpoint,
		Mode:                 configuration.SynchronizationMode,
		Interval:             time.Duration(configuration.SynchronizationIntervalSeconds) * time.Second,
		BackupRetentionCount: configuration.BackupRetentionCount,
	}, nil
}

func validateRemoteEndpoint(endpoint RemoteEndpoint) error {
	if endpoint.Protocol != RemoteProtocolFTP && endpoint.Protocol != RemoteProtocolSFTP {
		return fmt.Errorf("remote destination protocol must be ftp or sftp")
	}
	if endpoint.Host == "" || endpoint.Port < 1 || endpoint.Port > 65535 {
		return fmt.Errorf("remote destination server and valid port are required")
	}
	if endpoint.Host != strings.ToLower(endpoint.Host) || containsRemoteHostDelimiter(endpoint.Host) {
		return fmt.Errorf("remote destination server is invalid")
	}
	if _, err := validateRemoteDirectoryPath(endpoint.Path); err != nil {
		return err
	}
	if err := validateRemotePathEscaping(endpoint.Path); err != nil {
		return err
	}
	return nil
}

func validateRemoteWatchSource(sourceRoot string) (string, error) {
	entryInformation, err := os.Lstat(sourceRoot)
	if err != nil {
		return "", fmt.Errorf("inspect remote watch source directory: %w", err)
	}
	if entryInformation.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("remote watch source directory must not be a symbolic link")
	}
	if !entryInformation.IsDir() {
		return "", fmt.Errorf("remote watch source is not a directory")
	}
	if err := scanRootForUnsupportedEntries(sourceRoot); err != nil {
		return "", fmt.Errorf("inspect remote watch source entries: %w", err)
	}
	if _, err := buildDirectoryInventory(sourceRoot); err != nil {
		return "", fmt.Errorf("read remote watch source contents: %w", err)
	}
	canonicalSourceRoot, err := canonicalizeBackupPath(sourceRoot)
	if err != nil {
		return "", fmt.Errorf("resolve remote watch source path: %w", err)
	}
	return canonicalSourceRoot, nil
}

func containsRemoteHostDelimiter(host string) bool {
	for _, character := range host {
		if character == '/' || character == '\\' || character == '@' || character == '?' || character == '#' || character <= ' ' {
			return true
		}
	}
	return false
}
