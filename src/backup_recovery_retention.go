package gosync

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BackupDestinationInspectionStatus string

const (
	BackupDestinationInspectionMissing BackupDestinationInspectionStatus = "missing"
	BackupDestinationInspectionEmpty   BackupDestinationInspectionStatus = "empty-unbound"
	BackupDestinationInspectionBound   BackupDestinationInspectionStatus = "bound"
	BackupDestinationInspectionForeign BackupDestinationInspectionStatus = "foreign"
	BackupDestinationInspectionDamaged BackupDestinationInspectionStatus = "damaged"
)

type BackupDestinationEntryKind string

const (
	BackupDestinationEntryConfirmed      BackupDestinationEntryKind = "confirmed"
	BackupDestinationEntryPendingRemoval BackupDestinationEntryKind = "pending-removal"
	BackupDestinationEntryOwnedCandidate BackupDestinationEntryKind = "owned-candidate"
	BackupDestinationEntryForeign        BackupDestinationEntryKind = "foreign"
)

type BackupDestinationEntry struct {
	Name string
	Kind BackupDestinationEntryKind
}

type BackupDestinationInspection struct {
	Status  BackupDestinationInspectionStatus
	Entries []BackupDestinationEntry
}

func inspectBackupSetDestination(store BackupSetStore, sourceRoot string) (BackupDestinationInspection, error) {
	status, err := inspectBackupDestination(store.DestinationPath())
	if err != nil {
		return BackupDestinationInspection{}, err
	}
	if status == BackupDestinationMissing {
		return BackupDestinationInspection{Status: BackupDestinationInspectionMissing}, nil
	}

	entries, err := os.ReadDir(store.DestinationPath())
	if err != nil {
		return BackupDestinationInspection{}, fmt.Errorf("read backup destination entries: %w", err)
	}
	loadResult, loadErr := store.Load()
	if loadErr != nil && loadResult.Status != BackupSetLoadMissing {
		return BackupDestinationInspection{Status: BackupDestinationInspectionDamaged}, loadErr
	}
	if loadResult.Status == BackupSetLoadMissing && len(entries) == 0 {
		return BackupDestinationInspection{Status: BackupDestinationInspectionEmpty}, nil
	}

	canonicalSource, err := canonicalizeBackupPath(sourceRoot)
	if err != nil {
		return BackupDestinationInspection{}, err
	}
	state := loadResult.State
	if loadResult.Status == BackupSetLoadValid {
		if err := validateBackupSetSourceBinding(state, canonicalSource); err != nil {
			return BackupDestinationInspection{Status: BackupDestinationInspectionDamaged}, err
		}
	}

	stateArchives := make(map[string]BackupArchiveRecord, len(state.Archives))
	inspectionEntries := make([]BackupDestinationEntry, 0, len(entries))
	damaged := false
	var firstDamage error
	if loadResult.Status == BackupSetLoadValid {
		for _, archive := range state.Archives {
			stateArchives[archive.FileName] = archive
			archivePath := filepath.Join(store.DestinationPath(), archive.FileName)
			if err := validateBackupArchiveRecord(archivePath, archive, state.SourcePath, store.DestinationPath()); err != nil {
				damaged = true
				if firstDamage == nil {
					firstDamage = err
				}
				continue
			}
			kind := BackupDestinationEntryConfirmed
			if archive.Lifecycle == BackupArchivePendingRemoval {
				kind = BackupDestinationEntryPendingRemoval
			}
			inspectionEntries = append(inspectionEntries, BackupDestinationEntry{Name: archive.FileName, Kind: kind})
		}
	}

	foreign := false
	for _, directoryEntry := range entries {
		name := directoryEntry.Name()
		if _, known := stateArchives[name]; known {
			continue
		}
		entryPath := filepath.Join(store.DestinationPath(), name)
		candidate, candidateErr := isOwnedBackupCandidate(entryPath, canonicalSource, store.DestinationPath())
		if candidateErr == nil && candidate {
			inspectionEntries = append(inspectionEntries, BackupDestinationEntry{Name: name, Kind: BackupDestinationEntryOwnedCandidate})
			continue
		}
		inspectionEntries = append(inspectionEntries, BackupDestinationEntry{Name: name, Kind: BackupDestinationEntryForeign})
		foreign = true
	}
	sort.Slice(inspectionEntries, func(first, second int) bool { return inspectionEntries[first].Name < inspectionEntries[second].Name })
	if damaged {
		return BackupDestinationInspection{Status: BackupDestinationInspectionDamaged, Entries: inspectionEntries}, firstDamage
	}
	if foreign {
		return BackupDestinationInspection{Status: BackupDestinationInspectionForeign, Entries: inspectionEntries}, errors.New("backup destination contains foreign entries")
	}
	return BackupDestinationInspection{Status: BackupDestinationInspectionBound, Entries: inspectionEntries}, nil
}

func validateBackupArchiveRecord(archivePath string, record BackupArchiveRecord, expectedSource, expectedDestination string) error {
	if err := verifyBackupArchiveDigest(archivePath, record.WholeArchiveDigest); err != nil {
		return fmt.Errorf("validate confirmed backup archive %q: %w", record.FileName, err)
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open confirmed backup archive %q: %w", record.FileName, err)
	}
	defer reader.Close()
	identity, err := decodeBackupArchiveIdentity(reader.Comment)
	if err != nil {
		return fmt.Errorf("decode confirmed backup archive %q: %w", record.FileName, err)
	}
	if err := validateBackupArchiveIdentity(identity, expectedSource, expectedDestination, record.ConfirmationOrder, record.InventoryDigest); err != nil {
		return fmt.Errorf("validate confirmed backup archive %q identity: %w", record.FileName, err)
	}
	if identity.ArchiveID != record.ArchiveID {
		return fmt.Errorf("confirmed backup archive %q identifier does not match state", record.FileName)
	}
	for _, file := range reader.File {
		archiveEntry, err := file.Open()
		if err != nil {
			return fmt.Errorf("open confirmed backup archive entry %q: %w", file.Name, err)
		}
		_, copyErr := io.Copy(io.Discard, archiveEntry)
		closeErr := archiveEntry.Close()
		if copyErr != nil {
			return fmt.Errorf("read confirmed backup archive entry %q: %w", file.Name, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close confirmed backup archive entry %q: %w", file.Name, closeErr)
		}
	}
	return nil
}

func isOwnedBackupCandidate(path string, expectedSource, expectedDestination string) (bool, error) {
	entryInfo, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !entryInfo.Mode().IsRegular() || filepath.Ext(path) != ".zip" {
		return false, nil
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false, nil
	}
	defer reader.Close()
	identity, err := decodeBackupArchiveIdentity(reader.Comment)
	if err != nil {
		return false, nil
	}
	if err := validateBackupArchiveIdentity(identity, expectedSource, expectedDestination, 0, ""); err != nil {
		return false, nil
	}
	return true, nil
}

func validateBackupSetSourceBinding(state BackupSetState, sourceRoot string) error {
	canonicalSource, err := canonicalizeBackupPath(sourceRoot)
	if err != nil {
		return err
	}
	if state.SourcePath == "" {
		return errors.New("backup set source is not bound")
	}
	if state.SourcePath != canonicalSource {
		return fmt.Errorf("backup set source %q does not match %q", state.SourcePath, canonicalSource)
	}
	return nil
}

func bindBackupSetSource(state BackupSetState, sourceRoot string) (BackupSetState, error) {
	canonicalSource, err := canonicalizeBackupPath(sourceRoot)
	if err != nil {
		return BackupSetState{}, err
	}
	if state.SourcePath != "" && state.SourcePath != canonicalSource {
		return BackupSetState{}, fmt.Errorf("backup set source %q does not match %q", state.SourcePath, canonicalSource)
	}
	state.SourcePath = canonicalSource
	if err := validateBackupSetState(state); err != nil {
		return BackupSetState{}, err
	}
	return state, nil
}

func recoverOwnedBackupCandidates(store BackupSetStore, sourceRoot string) error {
	inspection, err := inspectBackupSetDestination(store, sourceRoot)
	if err != nil {
		return err
	}
	for _, entry := range inspection.Entries {
		if entry.Kind != BackupDestinationEntryOwnedCandidate {
			continue
		}
		if err := os.Remove(filepath.Join(store.DestinationPath(), entry.Name)); err != nil {
			return fmt.Errorf("remove owned backup candidate %q: %w", entry.Name, err)
		}
	}
	return nil
}

func createBackupArchiveCandidate(destinationDirectory, archiveID string) (string, error) {
	if archiveID == "" || strings.ContainsAny(archiveID, `/\\`) {
		return "", errors.New("backup archive candidate identifier is invalid")
	}
	file, err := os.CreateTemp(destinationDirectory, ".gosync-backup-"+archiveID+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create backup archive candidate: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close backup archive candidate: %w", err)
	}
	return path, nil
}

func verifyBackupSourceStability(initial, archived, final BackupLogicalInventory) error {
	if !backupLogicalInventoriesEqual(initial, archived) || !backupLogicalInventoriesEqual(initial, final) {
		return errors.New("backup source changed during archive creation")
	}
	return nil
}

func publishBackupArchive(candidatePath, finalPath string) error {
	if filepath.Clean(filepath.Dir(candidatePath)) != filepath.Clean(filepath.Dir(finalPath)) {
		return errors.New("backup archive candidate and final path must share a directory")
	}
	if _, err := os.Lstat(finalPath); err == nil {
		return errors.New("backup archive final path already exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect backup archive final path: %w", err)
	}
	if err := os.Rename(candidatePath, finalPath); err != nil {
		return fmt.Errorf("publish backup archive: %w", err)
	}
	return nil
}

func confirmBackupArchive(store BackupSetStore, state BackupSetState, identity BackupArchiveIdentity, archivePath, archiveDigest string) (BackupSetState, error) {
	if identity.DestinationPath != store.DestinationPath() || identity.ConfirmationOrder != state.NextConfirmationOrder {
		return BackupSetState{}, errors.New("backup archive confirmation identity does not match state")
	}
	if !isBackupSHA256Digest(archiveDigest) {
		return BackupSetState{}, errors.New("backup archive confirmation digest is invalid")
	}
	canonicalArchivePath, err := canonicalizeBackupPath(archivePath)
	if err != nil {
		return BackupSetState{}, fmt.Errorf("canonicalize backup archive confirmation path: %w", err)
	}
	if filepath.Clean(filepath.Dir(canonicalArchivePath)) != filepath.Clean(store.DestinationPath()) {
		return BackupSetState{}, errors.New("backup archive confirmation path is outside destination")
	}
	boundState, err := bindBackupSetSource(state, identity.SourcePath)
	if err != nil {
		return BackupSetState{}, err
	}
	fileName := filepath.Base(canonicalArchivePath)
	for _, archive := range boundState.Archives {
		if archive.ArchiveID == identity.ArchiveID || archive.FileName == fileName {
			return BackupSetState{}, errors.New("backup archive is already confirmed")
		}
	}
	boundState.Archives = append(boundState.Archives, BackupArchiveRecord{
		ArchiveID:          identity.ArchiveID,
		FileName:           fileName,
		ConfirmationOrder:  identity.ConfirmationOrder,
		CreatedAt:          identity.CreatedAt,
		WholeArchiveDigest: archiveDigest,
		InventoryDigest:    identity.InventoryDigest,
		Lifecycle:          BackupArchiveConfirmed,
	})
	boundState.NextConfirmationOrder++
	if err := store.Save(boundState); err != nil {
		return BackupSetState{}, fmt.Errorf("commit backup archive confirmation: %w", err)
	}
	return boundState, nil
}

func rollbackBackupFirstCycle(destinationDirectory, candidatePath string) error {
	var rollbackErrors []error
	if candidatePath != "" {
		if err := os.Remove(candidatePath); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove backup candidate during rollback: %w", err))
		}
	}
	entries, err := os.ReadDir(destinationDirectory)
	if err != nil && !os.IsNotExist(err) {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("inspect backup destination during rollback: %w", err))
	} else if err == nil && len(entries) == 0 {
		if err := os.Remove(destinationDirectory); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove backup destination during rollback: %w", err))
		}
	}
	return errors.Join(rollbackErrors...)
}

func planBackupRetention(state BackupSetState, retentionCount int) ([]BackupArchiveRecord, error) {
	if retentionCount < MinimumBackupRetentionCount || retentionCount > MaximumBackupRetentionCount {
		return nil, fmt.Errorf("backup retention must be between %d and %d versions", MinimumBackupRetentionCount, MaximumBackupRetentionCount)
	}
	if state.PendingRemoval != nil {
		return nil, errors.New("backup retention has a pending removal")
	}
	confirmed := make([]BackupArchiveRecord, 0, len(state.Archives))
	for _, archive := range state.Archives {
		if archive.Lifecycle == BackupArchiveConfirmed {
			confirmed = append(confirmed, archive)
		}
	}
	sort.Slice(confirmed, func(first, second int) bool {
		return confirmed[first].ConfirmationOrder < confirmed[second].ConfirmationOrder
	})
	if len(confirmed) <= retentionCount {
		return []BackupArchiveRecord{}, nil
	}
	return confirmed[:len(confirmed)-retentionCount], nil
}

func markBackupArchivePendingRemoval(store BackupSetStore, state BackupSetState, record BackupArchiveRecord) (BackupSetState, error) {
	updated := state
	updated.Archives = append([]BackupArchiveRecord(nil), state.Archives...)
	found := false
	for index := range updated.Archives {
		archive := &updated.Archives[index]
		if archive.ArchiveID == record.ArchiveID && archive.FileName == record.FileName && archive.ConfirmationOrder == record.ConfirmationOrder && archive.Lifecycle == BackupArchiveConfirmed {
			archive.Lifecycle = BackupArchivePendingRemoval
			found = true
		}
	}
	if !found {
		return BackupSetState{}, errors.New("backup archive to remove was not confirmed")
	}
	updated.PendingRemoval = &BackupRemovalState{ArchiveID: record.ArchiveID, FileName: record.FileName, ConfirmationOrder: record.ConfirmationOrder}
	if err := store.Save(updated); err != nil {
		return BackupSetState{}, fmt.Errorf("mark backup archive pending removal: %w", err)
	}
	return updated, nil
}

func resumeBackupPendingRemoval(store BackupSetStore, state BackupSetState) (BackupSetState, error) {
	if state.PendingRemoval == nil {
		return state, nil
	}
	archivePath := filepath.Join(store.DestinationPath(), state.PendingRemoval.FileName)
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return state, fmt.Errorf("remove pending backup archive %q: %w", state.PendingRemoval.FileName, err)
	}
	updated := state
	updated.Archives = make([]BackupArchiveRecord, 0, len(state.Archives)-1)
	for _, archive := range state.Archives {
		if archive.ArchiveID != state.PendingRemoval.ArchiveID {
			updated.Archives = append(updated.Archives, archive)
		}
	}
	updated.PendingRemoval = nil
	if err := store.Save(updated); err != nil {
		return state, fmt.Errorf("complete pending backup removal: %w", err)
	}
	return updated, nil
}

func executeBackupRetention(store BackupSetStore, state BackupSetState, retentionCount int) error {
	if state.PendingRemoval != nil {
		var err error
		state, err = resumeBackupPendingRemoval(store, state)
		if err != nil {
			return err
		}
	}
	expired, err := planBackupRetention(state, retentionCount)
	if err != nil {
		return err
	}
	for _, record := range expired {
		state, err = markBackupArchivePendingRemoval(store, state, record)
		if err != nil {
			return err
		}
		state, err = resumeBackupPendingRemoval(store, state)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateBackupRetentionCompliance(state BackupSetState, retentionCount int) error {
	if retentionCount < MinimumBackupRetentionCount || retentionCount > MaximumBackupRetentionCount {
		return fmt.Errorf("backup retention must be between %d and %d versions", MinimumBackupRetentionCount, MaximumBackupRetentionCount)
	}
	if state.PendingRemoval != nil {
		return errors.New("backup retention has an unfinished removal")
	}
	confirmed := 0
	for _, archive := range state.Archives {
		if archive.Lifecycle == BackupArchiveConfirmed {
			confirmed++
		}
	}
	if confirmed > retentionCount {
		return fmt.Errorf("backup retention limit is not satisfied: %d confirmed versions exceed %d", confirmed, retentionCount)
	}
	return nil
}
