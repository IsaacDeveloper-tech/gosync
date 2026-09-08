package gosync_test

import (
	"io/fs"

	gosync "gosync/src"
)

type EntryKind = gosync.EntryKind
type SynchronizationEntry = gosync.SynchronizationEntry
type EntryComparison = gosync.EntryComparison
type SynchronizationActionKind = gosync.SynchronizationActionKind
type SynchronizationAction = gosync.SynchronizationAction
type SynchronizationResult = gosync.SynchronizationResult
type RootPaths = gosync.RootPaths
type DirectoryInventory = gosync.DirectoryInventory
type ConfirmedSynchronizationState = gosync.ConfirmedSynchronizationState
type confirmedStateStore = gosync.ConfirmedStateStore
type SynchronizationSide = gosync.SynchronizationSide
type MissingStateRecovery = gosync.MissingStateRecovery
type SynchronizationPlan = gosync.SynchronizationPlan
type SynchronizationExecutionOptions = gosync.SynchronizationExecutionOptions
type WatchLoopOptions = gosync.WatchLoopOptions
type WatchCommandOptions = gosync.WatchCommandOptions

const (
	EntryKindFile                         = gosync.EntryKindFile
	EntryKindDirectory                    = gosync.EntryKindDirectory
	SynchronizationActionCopyToFirst      = gosync.SynchronizationActionCopyToFirst
	SynchronizationActionCopyToSecond     = gosync.SynchronizationActionCopyToSecond
	SynchronizationActionDeleteFromFirst  = gosync.SynchronizationActionDeleteFromFirst
	SynchronizationActionDeleteFromSecond = gosync.SynchronizationActionDeleteFromSecond
	SynchronizationSideFirst              = gosync.SynchronizationSideFirst
	SynchronizationSideSecond             = gosync.SynchronizationSideSecond
	watchUsage                            = gosync.WatchUsage
)

var errSynchronizationIncomplete = gosync.ErrSynchronizationIncomplete

var newConfirmedStateStore = gosync.NewConfirmedStateStore
var newConfirmedStateStoreAt = gosync.NewConfirmedStateStoreAt
var validateRootPaths = gosync.ValidateRootPaths
var normalizeRootPath = gosync.NormalizeRootPath
var rootContains = gosync.RootContains
var scanRootsForUnsupportedEntries = gosync.ScanRootsForUnsupportedEntries
var scanRootForUnsupportedEntries = gosync.ScanRootForUnsupportedEntries
var isSupportedEntryMode = func(mode fs.FileMode) bool { return gosync.IsSupportedEntryMode(mode) }
var ensureRootDirectories = gosync.EnsureRootDirectories
var buildDirectoryInventory = gosync.BuildDirectoryInventory
var compareDirectoryInventories = gosync.CompareDirectoryInventories
var directoryInventoriesHaveEquivalentContents = gosync.DirectoryInventoriesHaveEquivalentContents
var classifyAdditionsAndDeletions = gosync.ClassifyAdditionsAndDeletions
var recoverFromMissingConfirmedState = gosync.RecoverFromMissingConfirmedState
var resolveFileContentConflict = gosync.ResolveFileContentConflict
var resolveFileDirectoryConflict = gosync.ResolveFileDirectoryConflict
var generateSynchronizationPlan = gosync.GenerateSynchronizationPlan
var executeSynchronizationPlan = gosync.ExecuteSynchronizationPlan
var retryLockedFile = gosync.RetryLockedFile
var verifyAndCommitSynchronization = gosync.VerifyAndCommitSynchronization
var parseWatchCommand = gosync.ParseWatchCommand
var runWatchLoop = gosync.RunWatchLoop
var runWatchCommand = gosync.RunWatchCommand
var synchronizeDirectories = gosync.SynchronizeDirectories
