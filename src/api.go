package gosync

import (
	"io"
	"io/fs"
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
