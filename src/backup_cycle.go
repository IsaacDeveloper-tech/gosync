package gosync

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type BackupCycleOptions struct {
	ApplicationDataDirectory string
	Logger                   *LoggingCoordinator
	Clock                    func() time.Time
	ArchiveID                func() (string, error)
	Notify                   func(string)
}

func runBackupCycle(policy BackupPolicySnapshot, options BackupCycleOptions) (cycleErr error) {
	if options.ApplicationDataDirectory == "" {
		return errors.New("backup application-data directory is required")
	}
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	archiveIDGenerator := options.ArchiveID
	if archiveIDGenerator == nil {
		archiveIDGenerator = generateBackupArchiveID
	}
	if err := logBackupCycle(options.Logger, LogEventBackupCycleStarted, LogSeverityInfo, "backup cycle started", nil); err != nil {
		return err
	}

	destinationCreated := false
	confirmed := false
	candidatePath := ""
	defer func() {
		if cycleErr == nil {
			return
		}
		if !confirmed && destinationCreated {
			if rollbackErr := rollbackBackupFirstCycle(policy.Destination, candidatePath); rollbackErr != nil {
				cycleErr = errors.Join(cycleErr, rollbackErr)
			}
		}
		if logErr := logBackupCycle(options.Logger, LogEventBackupCycleFailed, LogSeverityError, "backup cycle failed", map[string]string{"error": cycleErr.Error()}); logErr != nil {
			cycleErr = errors.Join(cycleErr, logErr)
		}
	}()

	if err := validateBackupSource(policy.Source); err != nil {
		return err
	}
	destinationStatus, err := inspectBackupDestination(policy.Destination)
	if err != nil {
		return err
	}
	if destinationStatus == BackupDestinationMissing {
		if err := os.MkdirAll(policy.Destination, 0o700); err != nil {
			return fmt.Errorf("create backup destination: %w", err)
		}
		destinationCreated = true
	}
	backupStore, err := newBackupSetStoreAt(options.ApplicationDataDirectory, policy.Destination)
	if err != nil {
		return err
	}
	inspection, err := inspectBackupSetDestination(backupStore, policy.Source)
	if err != nil {
		return err
	}
	ownedCandidateFound := false
	for _, entry := range inspection.Entries {
		if entry.Kind == BackupDestinationEntryOwnedCandidate {
			ownedCandidateFound = true
			break
		}
	}
	if err := recoverOwnedBackupCandidates(backupStore, policy.Source); err != nil {
		if inspection.Status != BackupDestinationInspectionEmpty && inspection.Status != BackupDestinationInspectionMissing {
			return err
		}
	}
	if ownedCandidateFound {
		if err := logBackupCycle(options.Logger, LogEventBackupCandidateRecovered, LogSeverityInfo, "backup destination inspected", nil); err != nil {
			return err
		}
	}
	loadResult, loadErr := backupStore.Load()
	if loadErr != nil && loadResult.Status != BackupSetLoadMissing {
		return loadErr
	}
	var state BackupSetState
	if loadResult.Status == BackupSetLoadMissing {
		state = BackupSetState{SchemaVersion: BackupSetStateSchemaVersion, DestinationPath: backupStore.DestinationPath(), SourcePath: "", NextConfirmationOrder: 1, Archives: []BackupArchiveRecord{}}
	} else {
		state = loadResult.State
		if err := validateBackupSetSourceBinding(state, policy.Source); err != nil {
			return err
		}
	}
	if state.PendingRemoval != nil || len(state.Archives) > policy.BackupRetentionCount {
		if err := executeBackupRetention(backupStore, state, policy.BackupRetentionCount); err != nil {
			return err
		}
		loadResult, err = backupStore.Load()
		if err != nil || loadResult.Status != BackupSetLoadValid {
			if err == nil {
				err = errors.New("backup state unavailable after retention")
			}
			return err
		}
		state = loadResult.State
	}

	initialInventory, err := buildBackupLogicalInventory(policy.Source)
	if err != nil {
		return err
	}
	archiveID, err := archiveIDGenerator()
	if err != nil {
		return fmt.Errorf("generate backup archive identifier: %w", err)
	}
	createdAt := clock().UTC()
	identity := BackupArchiveIdentity{
		SchemaVersion:     BackupArchiveIdentitySchemaVersion,
		ArchiveID:         archiveID,
		SourcePath:        policy.Source,
		DestinationPath:   backupStore.DestinationPath(),
		CreatedAt:         createdAt,
		ConfirmationOrder: state.NextConfirmationOrder,
		InventoryDigest:   initialInventory.Digest,
	}
	candidatePath, err = createBackupArchiveCandidate(backupStore.DestinationPath(), archiveID)
	if err != nil {
		return err
	}
	if err := writeBackupArchive(candidatePath, policy.Source, initialInventory, identity); err != nil {
		return err
	}
	if err := verifyBackupArchive(candidatePath, identity, initialInventory); err != nil {
		return err
	}
	if err := logBackupCycle(options.Logger, LogEventBackupArchiveVerified, LogSeverityInfo, "backup archive verified", nil); err != nil {
		return err
	}
	finalInventory, err := buildBackupLogicalInventory(policy.Source)
	if err != nil {
		return err
	}
	if err := verifyBackupSourceStability(initialInventory, initialInventory, finalInventory); err != nil {
		return err
	}
	archiveName, err := generateBackupArchiveName(func() time.Time { return createdAt }, archiveID)
	if err != nil {
		return err
	}
	finalPath := filepath.Join(backupStore.DestinationPath(), archiveName)
	if err := publishBackupArchive(candidatePath, finalPath); err != nil {
		return err
	}
	candidatePath = ""
	archiveDigest, err := calculateBackupArchiveDigest(finalPath)
	if err != nil {
		return err
	}
	state, err = confirmBackupArchive(backupStore, state, identity, finalPath, archiveDigest)
	if err != nil {
		return err
	}
	confirmed = true
	if err := logBackupCycle(options.Logger, LogEventBackupArchiveConfirmed, LogSeverityInfo, "backup archive confirmed", nil); err != nil {
		return err
	}
	if err := executeBackupRetention(backupStore, state, policy.BackupRetentionCount); err != nil {
		return err
	}
	if err := logBackupCycle(options.Logger, LogEventBackupRetentionApplied, LogSeverityInfo, "backup retention applied", nil); err != nil {
		return err
	}
	if err := logBackupCycle(options.Logger, LogEventBackupCycleCompleted, LogSeverityInfo, "backup cycle completed successfully", nil); err != nil {
		return err
	}
	return nil
}

func generateBackupArchiveID() (string, error) {
	identifier := make([]byte, 16)
	if _, err := rand.Read(identifier); err != nil {
		return "", err
	}
	return hex.EncodeToString(identifier), nil
}

func logBackupCycle(logger *LoggingCoordinator, event LogEvent, severity LogSeverity, message string, context map[string]string) error {
	if logger == nil {
		return nil
	}
	return logger.Log(LogEntry{Severity: severity, Event: event, Message: message, Context: context})
}
