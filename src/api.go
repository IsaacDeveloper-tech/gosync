package gosync

import (
	"io"
	"io/fs"
	"time"
)

var ErrSynchronizationIncomplete = errSynchronizationIncomplete

const WatchUsage = watchUsage

func NewConfirmedStateStore() (ConfirmedStateStore, error) {
	return newConfirmedStateStore()
}

func NewConfirmedStateStoreAt(directory string) ConfirmedStateStore {
	return newConfirmedStateStoreAt(directory)
}

func ValidateRootPaths(firstRoot, secondRoot string) (RootPaths, error) {
	return validateRootPaths(firstRoot, secondRoot)
}

func NormalizeRootPath(root string) (string, error) {
	return normalizeRootPath(root)
}

func RootContains(parentRoot, candidateRoot string) (bool, error) {
	return rootContains(parentRoot, candidateRoot)
}

func ScanRootsForUnsupportedEntries(roots RootPaths) error {
	return scanRootsForUnsupportedEntries(roots)
}

func ScanRootForUnsupportedEntries(root string) error {
	return scanRootForUnsupportedEntries(root)
}

func IsSupportedEntryMode(mode fs.FileMode) bool {
	return isSupportedEntryMode(mode)
}

func EnsureRootDirectories(roots RootPaths) error {
	return ensureRootDirectories(roots)
}

func BuildDirectoryInventory(root string) (DirectoryInventory, error) {
	return buildDirectoryInventory(root)
}

func CompareDirectoryInventories(firstInventory, secondInventory DirectoryInventory) []EntryComparison {
	return compareDirectoryInventories(firstInventory, secondInventory)
}

func DirectoryInventoriesHaveEquivalentContents(firstInventory, secondInventory DirectoryInventory) bool {
	return directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory)
}

func ClassifyAdditionsAndDeletions(comparisons []EntryComparison) []SynchronizationAction {
	return classifyAdditionsAndDeletions(comparisons)
}

func RecoverFromMissingConfirmedState(
	confirmedStateFound bool,
	firstInventory DirectoryInventory,
	secondInventory DirectoryInventory,
	requestAuthoritativeSide func() (SynchronizationSide, error),
) (MissingStateRecovery, error) {
	return recoverFromMissingConfirmedState(confirmedStateFound, firstInventory, secondInventory, requestAuthoritativeSide)
}

func ResolveFileContentConflict(
	comparison EntryComparison,
	requestConflictSide func() (SynchronizationSide, error),
) (SynchronizationAction, bool, error) {
	return resolveFileContentConflict(comparison, requestConflictSide)
}

func ResolveFileDirectoryConflict(
	comparison EntryComparison,
	directoryContents []string,
	requestConflictSide func([]string) (SynchronizationSide, error),
) (SynchronizationAction, bool, error) {
	return resolveFileDirectoryConflict(comparison, directoryContents, requestConflictSide)
}

func GenerateSynchronizationPlan(
	comparisons []EntryComparison,
	directoryContents map[string][]string,
	requestFileConflictSide func(EntryComparison) (SynchronizationSide, error),
	requestFileDirectoryConflictSide func(EntryComparison, []string) (SynchronizationSide, error),
) (SynchronizationPlan, error) {
	return generateSynchronizationPlan(comparisons, directoryContents, requestFileConflictSide, requestFileDirectoryConflictSide)
}

func ExecuteSynchronizationPlan(roots RootPaths, plan SynchronizationPlan, options SynchronizationExecutionOptions) SynchronizationResult {
	return executeSynchronizationPlan(roots, plan, options)
}

func RetryLockedFile(filePath string, operation func() error, options SynchronizationExecutionOptions) error {
	return retryLockedFile(filePath, operation, options)
}

func VerifyAndCommitSynchronization(roots RootPaths, store ConfirmedStateStore, result SynchronizationResult) error {
	return verifyAndCommitSynchronization(roots, store, result)
}

func ParseWatchCommand(arguments []string) (RootPaths, error) {
	return parseWatchCommand(arguments)
}

func RunWatchLoop(synchronize func() error, options WatchLoopOptions) error {
	return runWatchLoop(synchronize, options)
}

func RunWatchCommand(arguments []string, options WatchCommandOptions) error {
	return runWatchCommand(arguments, options)
}

func SynchronizeDirectories(roots RootPaths, store ConfirmedStateStore, input io.Reader, output io.Writer) error {
	return synchronizeDirectories(roots, store, input, output)
}

func NewLogEntry(
	clock func() time.Time,
	severity LogSeverity,
	event LogEvent,
	message string,
	context map[string]string,
) LogEntry {
	return newLogEntry(clock, severity, event, message, context)
}

func FormatLogEntry(entry LogEntry) string {
	return formatLogEntry(entry)
}

func NewLogSanitizer() *LogSanitizer {
	return newLogSanitizer()
}

func NewConsoleDestination(writer io.Writer) ConsoleDestination {
	return newConsoleDestination(writer)
}

func ResolveLogDirectory() (string, error) {
	return resolveLogDirectory()
}

func LogLocationOverlapsRoots(logLocation string, roots RootPaths) (bool, error) {
	return logLocationOverlapsRoots(logLocation, roots)
}

func NewRotatingFileDestination(directory string) RotatingFileDestination {
	return newRotatingFileDestination(directory)
}

func NewLoggingCoordinator(options LoggingCoordinatorOptions) (*LoggingCoordinator, error) {
	return newLoggingCoordinator(options)
}

func InitializeLoggingCoordinator(roots RootPaths, options LoggingCoordinatorOptions) (*LoggingCoordinator, error) {
	return initializeLoggingCoordinator(roots, options)
}

func SynchronizeDirectoriesWithLogger(
	roots RootPaths,
	store ConfirmedStateStore,
	input io.Reader,
	output io.Writer,
	logger *LoggingCoordinator,
) error {
	return synchronizeDirectoriesWithLogger(roots, store, input, output, logger)
}

func ParseConfigureCommand(arguments []string) error {
	return parseConfigureCommand(arguments)
}

func ParseSynchronizationInterval(answer string) (int, error) {
	return parseSynchronizationInterval(answer)
}

func ParseSynchronizationMode(answer string) (SynchronizationMode, error) {
	return parseSynchronizationMode(answer)
}

func ParseBackupRetentionCount(answer string) (int, error) {
	return parseBackupRetentionCount(answer)
}

func CollectConfigurationDraft(input io.Reader, output io.Writer) (ConfigurationDraft, error) {
	return collectConfigurationDraft(input, output)
}

func ConfirmConfiguration(input io.Reader, output io.Writer, draft ConfigurationDraft) (bool, error) {
	return confirmConfiguration(input, output, draft)
}

func RunInteractiveConfiguration(input io.Reader, output io.Writer) (ConfigurationSnapshot, error) {
	return runInteractiveConfiguration(input, output)
}

func ResolveConfigurationFilePath() (string, error) {
	return resolveConfigurationFilePath()
}

func EncodeConfiguration(configuration ConfigurationSnapshot) (string, error) {
	return encodeConfiguration(configuration)
}

func DecodeConfiguration(encodedConfiguration string) (ConfigurationSnapshot, error) {
	return decodeConfiguration(encodedConfiguration)
}

func ValidateConfigurationSnapshot(configuration ConfigurationSnapshot) error {
	return validateConfigurationSnapshot(configuration)
}

func InspectConfigurationFile(path string) (ConfigurationFileStatus, error) {
	return inspectConfigurationFile(path)
}

func LoadConfiguration(path string) (ConfigurationLoadResult, error) {
	return loadConfiguration(path)
}

func NewConfigurationStore(path string) ConfigurationStore {
	return newConfigurationStore(path)
}

func AcquireConfigurationOwnership(configurationPath string) (ConfigurationOwnership, error) {
	return acquireConfigurationOwnership(configurationPath)
}

func NewConfigurationService(options ConfigurationServiceOptions) *ConfigurationService {
	return newConfigurationService(options)
}

func RunConfigureCommand(arguments []string, options ConfigureCommandOptions) error {
	return runConfigureCommand(arguments, options)
}

func BuildBackupPolicySnapshot(roots RootPaths, configuration ConfigurationSnapshot) (BackupPolicySnapshot, error) {
	return buildBackupPolicySnapshot(roots, configuration)
}

func DeriveBackupRootPaths(sourceRoot, destinationRoot string) (BackupRootPaths, error) {
	return deriveBackupRootPaths(sourceRoot, destinationRoot)
}

func ValidateBackupSource(sourceRoot string) error {
	return validateBackupSource(sourceRoot)
}

func InspectBackupDestination(destinationRoot string) (BackupDestinationStatus, error) {
	return inspectBackupDestination(destinationRoot)
}

func ValidateBackupRootPaths(sourceRoot, destinationRoot, configurationPath string) (BackupRootPaths, error) {
	return validateBackupRootPaths(sourceRoot, destinationRoot, configurationPath)
}

func ValidateBackupSetState(state BackupSetState) error {
	return validateBackupSetState(state)
}

func EncodeBackupSetState(state BackupSetState) (string, error) {
	return encodeBackupSetState(state)
}

func DecodeBackupSetState(encoded string) (BackupSetState, error) {
	return decodeBackupSetState(encoded)
}

func NewBackupSetStoreAt(applicationDataDirectory, destinationRoot string) (BackupSetStore, error) {
	return newBackupSetStoreAt(applicationDataDirectory, destinationRoot)
}

func EncodeBackupArchiveIdentity(identity BackupArchiveIdentity) (string, error) {
	return encodeBackupArchiveIdentity(identity)
}

func DecodeBackupArchiveIdentity(encoded string) (BackupArchiveIdentity, error) {
	return decodeBackupArchiveIdentity(encoded)
}

func ValidateBackupArchiveIdentity(identity BackupArchiveIdentity, expectedSource, expectedDestination string, expectedOrder uint64, expectedInventoryDigest string) error {
	return validateBackupArchiveIdentity(identity, expectedSource, expectedDestination, expectedOrder, expectedInventoryDigest)
}

func GenerateBackupArchiveName(clock func() time.Time, archiveID string) (string, error) {
	return generateBackupArchiveName(clock, archiveID)
}

func AcquireBackupDestinationOwnership(lockPath string) (BackupDestinationOwnership, error) {
	return acquireBackupDestinationOwnership(lockPath)
}

func WriteBackupArchive(path string, sourceRoot string, inventory BackupLogicalInventory, identity BackupArchiveIdentity) error {
	return writeBackupArchive(path, sourceRoot, inventory, identity)
}

func VerifyBackupArchive(archivePath string, expectedIdentity BackupArchiveIdentity, expectedInventory BackupLogicalInventory) error {
	return verifyBackupArchive(archivePath, expectedIdentity, expectedInventory)
}

func CalculateBackupArchiveDigest(archivePath string) (string, error) {
	return calculateBackupArchiveDigest(archivePath)
}

func VerifyBackupArchiveDigest(archivePath, expectedDigest string) error {
	return verifyBackupArchiveDigest(archivePath, expectedDigest)
}

func ValidateConfigurationPath(configurationPath string, roots RootPaths) error {
	return validateConfigurationPath(configurationPath, roots)
}

func ValidateUnidirectionalRoots(roots RootPaths) error {
	return validateUnidirectionalRoots(roots)
}

func SynchronizeUnidirectional(roots RootPaths, store ConfirmedStateStore, options SynchronizationExecutionOptions) error {
	return synchronizeUnidirectional(roots, store, options)
}

func GenerateUnidirectionalSynchronizationPlan(comparisons []EntryComparison) (SynchronizationPlan, error) {
	return generateUnidirectionalSynchronizationPlan(comparisons)
}
