package gosync

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RemoteEntryKind string

const (
	RemoteEntryRegular   RemoteEntryKind = "regular"
	RemoteEntryDirectory RemoteEntryKind = "directory"
	RemoteEntryLink      RemoteEntryKind = "link"
	RemoteEntrySpecial   RemoteEntryKind = "special"
	RemoteEntryUnknown   RemoteEntryKind = "unknown"
)

type RemoteRelativePath struct {
	value string
}

type RemoteEntry struct {
	Path RemoteRelativePath
	Kind RemoteEntryKind
}

type RemoteDirectoryInspection struct {
	RootExists bool
	RootKind   RemoteEntryKind
	Complete   bool
	Entries    []RemoteEntry
}

type RemoteCapabilities struct {
	CompleteTypedListings bool
	NonRedirectingPaths   bool
	LosslessNames         bool
	BinaryTransfers       bool
	ReadBackVerification  bool
	ExclusiveOwnership    bool
	DurableConfirmation   bool
}

type RemoteCapability string

const (
	RemoteCapabilityTypedListings        RemoteCapability = "complete typed listings"
	RemoteCapabilityNonRedirectingPaths  RemoteCapability = "non-redirecting selected paths"
	RemoteCapabilityLosslessNames        RemoteCapability = "lossless distinct path names"
	RemoteCapabilityBinaryTransfers      RemoteCapability = "binary-safe transfers"
	RemoteCapabilityReadBackVerification RemoteCapability = "independent read-back verification"
	RemoteCapabilityExclusiveOwnership   RemoteCapability = "exclusive cross-client ownership"
	RemoteCapabilityDurableConfirmation  RemoteCapability = "durable shared confirmation"
)

type RemoteCapabilityError struct {
	Mode    SynchronizationMode
	Missing []RemoteCapability
}

type RemoteProtocolOperations interface {
	InspectDirectory(selectedPath string) (RemoteDirectoryInspection, error)
	ReadFile(selectedPath string) (io.ReadCloser, error)
	WriteFile(selectedPath string, contents io.Reader) error
	CreateDirectory(selectedPath string) error
	DeleteFile(selectedPath string) error
	DeleteDirectory(selectedPath string) error
	Publish(candidatePath, finalPath string) error
	Capabilities() (RemoteCapabilities, error)
	Close() error
}

type RemoteDestination interface {
	Inspect() (RemoteDirectoryInspection, error)
	ReadFile(path RemoteRelativePath) (io.ReadCloser, error)
	WriteFile(path RemoteRelativePath, contents io.Reader) error
	CreateDirectory(path RemoteRelativePath) error
	DeleteFile(path RemoteRelativePath) error
	DeleteDirectory(path RemoteRelativePath) error
	Publish(candidatePath, finalPath RemoteRelativePath) error
	Capabilities() (RemoteCapabilities, error)
	Close() error
}

type ScopedRemoteDestination struct {
	selectedRoot string
	operations   RemoteProtocolOperations
}

type RemoteOperationFailure struct {
	OperationError error
	CleanupError   error
	CloseError     error
}

func newRemoteRelativePath(value string) (RemoteRelativePath, error) {
	if value == "" {
		return RemoteRelativePath{}, errors.New("remote relative path must not be empty; use the remote root for root operations")
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || !utf8.ValidString(value) {
		return RemoteRelativePath{}, errors.New("remote path must be a valid relative path using forward slashes")
	}
	if _, err := validateRemoteDirectoryPath("/" + value); err != nil {
		return RemoteRelativePath{}, fmt.Errorf("invalid remote relative path: %w", err)
	}
	if err := validateRemotePathEscaping("/" + value); err != nil {
		return RemoteRelativePath{}, fmt.Errorf("invalid remote relative path: %w", err)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return RemoteRelativePath{}, errors.New("remote relative path contains a control character")
		}
	}
	return RemoteRelativePath{value: value}, nil
}

func remoteRootPath() RemoteRelativePath {
	return RemoteRelativePath{}
}

func (remotePath RemoteRelativePath) String() string {
	return remotePath.value
}

func (remotePath RemoteRelativePath) IsRoot() bool {
	return remotePath.value == ""
}

func newScopedRemoteDestination(endpoint RemoteEndpoint, operations RemoteProtocolOperations) (*ScopedRemoteDestination, error) {
	if err := validateRemoteEndpoint(endpoint); err != nil {
		return nil, fmt.Errorf("validate scoped remote destination: %w", err)
	}
	if operations == nil {
		return nil, errors.New("remote protocol operations are required")
	}
	return &ScopedRemoteDestination{selectedRoot: endpoint.Path, operations: operations}, nil
}

func (destination *ScopedRemoteDestination) Inspect() (RemoteDirectoryInspection, error) {
	if err := destination.ensureOpen(); err != nil {
		return RemoteDirectoryInspection{}, err
	}
	inspection, err := destination.operations.InspectDirectory(destination.selectedRoot)
	if err != nil {
		return RemoteDirectoryInspection{}, fmt.Errorf("inspect selected remote directory: %w", err)
	}
	if err := validateRemoteDirectoryInspection(inspection); err != nil {
		return RemoteDirectoryInspection{}, err
	}
	return inspection, nil
}

func (destination *ScopedRemoteDestination) ReadFile(remotePath RemoteRelativePath) (io.ReadCloser, error) {
	selectedPath, err := destination.resolvePath(remotePath, false)
	if err != nil {
		return nil, err
	}
	reader, err := destination.operations.ReadFile(selectedPath)
	if err != nil {
		return nil, fmt.Errorf("read remote file %q: %w", remotePath, err)
	}
	if reader == nil {
		return nil, errors.New("remote read returned no file stream")
	}
	return reader, nil
}

func (destination *ScopedRemoteDestination) WriteFile(remotePath RemoteRelativePath, contents io.Reader) error {
	if contents == nil {
		return errors.New("remote file contents are required")
	}
	selectedPath, err := destination.resolvePath(remotePath, false)
	if err != nil {
		return err
	}
	if err := destination.operations.WriteFile(selectedPath, contents); err != nil {
		return fmt.Errorf("write remote file %q: %w", remotePath, err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) CreateDirectory(remotePath RemoteRelativePath) error {
	selectedPath, err := destination.resolvePath(remotePath, true)
	if err != nil {
		return err
	}
	if err := destination.operations.CreateDirectory(selectedPath); err != nil {
		return fmt.Errorf("create remote directory %q: %w", remotePath, err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) DeleteFile(remotePath RemoteRelativePath) error {
	selectedPath, err := destination.resolvePath(remotePath, false)
	if err != nil {
		return err
	}
	if err := destination.operations.DeleteFile(selectedPath); err != nil {
		return fmt.Errorf("delete remote file %q: %w", remotePath, err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) DeleteDirectory(remotePath RemoteRelativePath) error {
	selectedPath, err := destination.resolvePath(remotePath, false)
	if err != nil {
		return err
	}
	if err := destination.operations.DeleteDirectory(selectedPath); err != nil {
		return fmt.Errorf("delete remote directory %q: %w", remotePath, err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) Publish(candidatePath, finalPath RemoteRelativePath) error {
	selectedCandidatePath, err := destination.resolvePath(candidatePath, false)
	if err != nil {
		return fmt.Errorf("resolve remote publication candidate: %w", err)
	}
	selectedFinalPath, err := destination.resolvePath(finalPath, false)
	if err != nil {
		return fmt.Errorf("resolve remote publication destination: %w", err)
	}
	if selectedCandidatePath == selectedFinalPath {
		return errors.New("remote publication candidate and final paths must be distinct")
	}
	if err := destination.operations.Publish(selectedCandidatePath, selectedFinalPath); err != nil {
		return fmt.Errorf("publish remote file: %w", err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) Capabilities() (RemoteCapabilities, error) {
	if err := destination.ensureOpen(); err != nil {
		return RemoteCapabilities{}, err
	}
	capabilities, err := destination.operations.Capabilities()
	if err != nil {
		return RemoteCapabilities{}, fmt.Errorf("inspect remote server capabilities: %w", err)
	}
	return capabilities, nil
}

func (destination *ScopedRemoteDestination) Close() error {
	if destination == nil || destination.operations == nil {
		return errors.New("remote destination is unavailable")
	}
	if err := destination.operations.Close(); err != nil {
		return fmt.Errorf("close remote destination: %w", err)
	}
	return nil
}

func (destination *ScopedRemoteDestination) resolvePath(remotePath RemoteRelativePath, allowRoot bool) (string, error) {
	if err := destination.ensureOpen(); err != nil {
		return "", err
	}
	if remotePath.value == "" {
		if allowRoot {
			return destination.selectedRoot, nil
		}
		return "", errors.New("remote operation cannot replace or delete the selected root")
	}
	validatedPath, err := newRemoteRelativePath(remotePath.value)
	if err != nil {
		return "", err
	}
	selectedPath := path.Join(destination.selectedRoot, validatedPath.value)
	selectedPrefix := strings.TrimSuffix(destination.selectedRoot, "/") + "/"
	if !strings.HasPrefix(selectedPath, selectedPrefix) {
		return "", errors.New("remote operation escapes the selected directory")
	}
	return selectedPath, nil
}

func (destination *ScopedRemoteDestination) ensureOpen() error {
	if destination == nil || destination.operations == nil {
		return errors.New("remote destination is unavailable")
	}
	return nil
}

func validateRemoteDirectoryInspection(inspection RemoteDirectoryInspection) error {
	if !inspection.RootExists {
		if inspection.RootKind != "" || len(inspection.Entries) != 0 {
			return errors.New("missing remote destination cannot contain a root type or entries")
		}
		return nil
	}
	if !validRemoteEntryKind(inspection.RootKind) {
		return fmt.Errorf("remote root has invalid entry type %q", inspection.RootKind)
	}
	seenPaths := make(map[string]struct{}, len(inspection.Entries))
	for _, entry := range inspection.Entries {
		validatedPath, err := newRemoteRelativePath(entry.Path.value)
		if err != nil {
			return fmt.Errorf("remote listing contains an invalid relative path: %w", err)
		}
		if !validRemoteEntryKind(entry.Kind) {
			return fmt.Errorf("remote entry %q has invalid type %q", validatedPath, entry.Kind)
		}
		if _, found := seenPaths[validatedPath.value]; found {
			return fmt.Errorf("remote listing contains duplicate path %q", validatedPath)
		}
		seenPaths[validatedPath.value] = struct{}{}
	}
	return nil
}

func validRemoteEntryKind(entryKind RemoteEntryKind) bool {
	switch entryKind {
	case RemoteEntryRegular, RemoteEntryDirectory, RemoteEntryLink, RemoteEntrySpecial, RemoteEntryUnknown:
		return true
	default:
		return false
	}
}

func (capabilities RemoteCapabilities) MissingFor(mode SynchronizationMode) []RemoteCapability {
	missing := make([]RemoteCapability, 0)
	if !capabilities.CompleteTypedListings {
		missing = append(missing, RemoteCapabilityTypedListings)
	}
	if !capabilities.NonRedirectingPaths {
		missing = append(missing, RemoteCapabilityNonRedirectingPaths)
	}
	if !capabilities.LosslessNames {
		missing = append(missing, RemoteCapabilityLosslessNames)
	}
	if !capabilities.BinaryTransfers {
		missing = append(missing, RemoteCapabilityBinaryTransfers)
	}
	if !capabilities.ReadBackVerification {
		missing = append(missing, RemoteCapabilityReadBackVerification)
	}
	if mode == SynchronizationModeBackup {
		if !capabilities.ExclusiveOwnership {
			missing = append(missing, RemoteCapabilityExclusiveOwnership)
		}
		if !capabilities.DurableConfirmation {
			missing = append(missing, RemoteCapabilityDurableConfirmation)
		}
	}
	return missing
}

func requireRemoteCapabilities(capabilities RemoteCapabilities, mode SynchronizationMode) error {
	if mode != SynchronizationModeUnidirectional && mode != SynchronizationModeBackup {
		return fmt.Errorf("remote capabilities do not support synchronization mode %q", mode)
	}
	missing := capabilities.MissingFor(mode)
	if len(missing) == 0 {
		return nil
	}
	return &RemoteCapabilityError{Mode: mode, Missing: missing}
}

func (capabilityError *RemoteCapabilityError) Error() string {
	if capabilityError == nil {
		return "remote server capabilities are insufficient"
	}
	missingCapabilities := make([]string, 0, len(capabilityError.Missing))
	for _, missingCapability := range capabilityError.Missing {
		missingCapabilities = append(missingCapabilities, string(missingCapability))
	}
	return fmt.Sprintf("remote server lacks required capabilities for %s mode: %s", capabilityError.Mode, strings.Join(missingCapabilities, ", "))
}

func combineRemoteOperationErrors(operationError, cleanupError, closeError error) error {
	if operationError == nil && cleanupError == nil && closeError == nil {
		return nil
	}
	return &RemoteOperationFailure{OperationError: operationError, CleanupError: cleanupError, CloseError: closeError}
}

func (failure *RemoteOperationFailure) Error() string {
	if failure == nil {
		return ""
	}
	messages := make([]string, 0, 3)
	if failure.OperationError != nil {
		messages = append(messages, "operation: "+failure.OperationError.Error())
	}
	if failure.CleanupError != nil {
		messages = append(messages, "cleanup: "+failure.CleanupError.Error())
	}
	if failure.CloseError != nil {
		messages = append(messages, "close: "+failure.CloseError.Error())
	}
	return strings.Join(messages, "; ")
}

func (failure *RemoteOperationFailure) Unwrap() []error {
	if failure == nil {
		return nil
	}
	causes := make([]error, 0, 3)
	if failure.OperationError != nil {
		causes = append(causes, failure.OperationError)
	}
	if failure.CleanupError != nil {
		causes = append(causes, failure.CleanupError)
	}
	if failure.CloseError != nil {
		causes = append(causes, failure.CloseError)
	}
	return causes
}
