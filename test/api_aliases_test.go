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
type LogSeverity = gosync.LogSeverity
type LogEvent = gosync.LogEvent
type LogEntry = gosync.LogEntry
type LogSanitizer = gosync.LogSanitizer
type ConsoleDestination = gosync.ConsoleDestination
type RotatingFileDestination = gosync.RotatingFileDestination
type LoggingCoordinator = gosync.LoggingCoordinator
type LoggingCoordinatorOptions = gosync.LoggingCoordinatorOptions
type ConfigurationSnapshot = gosync.ConfigurationSnapshot
type ConfigurationDraft = gosync.ConfigurationDraft
type SynchronizationMode = gosync.SynchronizationMode

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
	LogSeverityDebug                      = gosync.LogSeverityDebug
	LogSeverityInfo                       = gosync.LogSeverityInfo
	LogSeverityWarn                       = gosync.LogSeverityWarn
	LogSeverityError                      = gosync.LogSeverityError
	LogEventSynchronizationStarted        = gosync.LogEventSynchronizationStarted
	LogEventSynchronizationCompleted      = gosync.LogEventSynchronizationCompleted
	LogEventChangeSynchronized            = gosync.LogEventChangeSynchronized
	LogEventConflictDetected              = gosync.LogEventConflictDetected
	LogEventRetryStarted                  = gosync.LogEventRetryStarted
	LogEventRetryCompleted                = gosync.LogEventRetryCompleted
	LogEventWarningRaised                 = gosync.LogEventWarningRaised
	LogEventOperationFailed               = gosync.LogEventOperationFailed
	LogEventPersistentDestinationFailure  = gosync.LogEventPersistentDestinationFailure
	LogEventConsoleDestinationFailure     = gosync.LogEventConsoleDestinationFailure
	ConfigurationSchemaVersion            = gosync.ConfigurationSchemaVersion
	MinimumSynchronizationIntervalSeconds = gosync.MinimumSynchronizationIntervalSeconds
	MaximumSynchronizationIntervalSeconds = gosync.MaximumSynchronizationIntervalSeconds
	SynchronizationModeBidirectional      = gosync.SynchronizationModeBidirectional
	SynchronizationModeUnidirectional     = gosync.SynchronizationModeUnidirectional
	SynchronizationModeBackup             = gosync.SynchronizationModeBackup
)

var errSynchronizationIncomplete = gosync.ErrSynchronizationIncomplete
var errLoggingUnavailable = gosync.ErrLoggingUnavailable
var errConfigurationCancelled = gosync.ErrConfigurationCancelled

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
var newLogEntry = gosync.NewLogEntry
var formatLogEntry = gosync.FormatLogEntry
var newLogSanitizer = gosync.NewLogSanitizer
var newConsoleDestination = gosync.NewConsoleDestination
var resolveLogDirectory = gosync.ResolveLogDirectory
var logLocationOverlapsRoots = gosync.LogLocationOverlapsRoots
var newRotatingFileDestination = gosync.NewRotatingFileDestination
var newLoggingCoordinator = gosync.NewLoggingCoordinator
var initializeLoggingCoordinator = gosync.InitializeLoggingCoordinator
var synchronizeDirectoriesWithLogger = gosync.SynchronizeDirectoriesWithLogger
var parseConfigureCommand = gosync.ParseConfigureCommand
var parseSynchronizationInterval = gosync.ParseSynchronizationInterval
var parseSynchronizationMode = gosync.ParseSynchronizationMode
var collectConfigurationDraft = gosync.CollectConfigurationDraft
var confirmConfiguration = gosync.ConfirmConfiguration
var runInteractiveConfiguration = gosync.RunInteractiveConfiguration
var resolveConfigurationFilePath = gosync.ResolveConfigurationFilePath
var encodeConfiguration = gosync.EncodeConfiguration
var decodeConfiguration = gosync.DecodeConfiguration
