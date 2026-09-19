package gosync_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectBackupDestinationReturnsMissingEmptyAndBoundWithoutMutation(t *testing.T) {
	root := t.TempDir()
	applicationData := filepath.Join(root, "application-data")
	if err := os.Mkdir(applicationData, 0o700); err != nil {
		t.Fatalf("Mkdir(application data) error = %v", err)
	}
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}

	missingDestination := filepath.Join(root, "missing")
	missingStore, err := newBackupSetStoreAt(applicationData, missingDestination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(missing) error = %v", err)
	}
	inspection, err := inspectBackupSetDestination(missingStore, source)
	if err != nil || inspection.Status != BackupDestinationInspectionMissing {
		t.Fatalf("inspect missing = %+v, %v, want missing without error", inspection, err)
	}
	if _, err := os.Stat(missingDestination); !os.IsNotExist(err) {
		t.Fatalf("missing destination stat error = %v, want no creation", err)
	}

	emptyDestination := filepath.Join(root, "empty")
	if err := os.Mkdir(emptyDestination, 0o700); err != nil {
		t.Fatalf("Mkdir(empty destination) error = %v", err)
	}
	emptyStore, err := newBackupSetStoreAt(applicationData, emptyDestination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(empty) error = %v", err)
	}
	inspection, err = inspectBackupSetDestination(emptyStore, source)
	if err != nil || inspection.Status != BackupDestinationInspectionEmpty {
		t.Fatalf("inspect empty = %+v, %v, want empty without error", inspection, err)
	}

	boundState := BackupSetState{SchemaVersion: 1, DestinationPath: emptyStore.DestinationPath(), SourcePath: storeSourcePath(t, source), NextConfirmationOrder: 1, Archives: []BackupArchiveRecord{}}
	if err := emptyStore.Save(boundState); err != nil {
		t.Fatalf("Save(bound state) error = %v", err)
	}
	inspection, err = inspectBackupSetDestination(emptyStore, source)
	if err != nil || inspection.Status != BackupDestinationInspectionBound {
		t.Fatalf("inspect bound = %+v, %v, want bound without error", inspection, err)
	}
}

func TestBackupSetSourceBindingAcceptsCanonicalAliasesAndRejectsDifferentSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	store, err := newBackupSetStoreAt(filepath.Join(root, "application-data"), destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	state := BackupSetState{SchemaVersion: 1, DestinationPath: store.DestinationPath(), SourcePath: "", NextConfirmationOrder: 1, Archives: []BackupArchiveRecord{}}
	bound, err := bindBackupSetSource(state, source)
	if err != nil {
		t.Fatalf("bindBackupSetSource() error = %v", err)
	}
	if bound.SourcePath == "" {
		t.Fatal("bound source path is empty")
	}
	alias := filepath.Join(root, "source-alias")
	if err := os.Symlink(source, alias); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := validateBackupSetSourceBinding(bound, alias); err != nil {
		t.Fatalf("validateBackupSetSourceBinding(alias) error = %v, want same canonical source", err)
	}
	if err := validateBackupSetSourceBinding(bound, filepath.Join(root, "other-source")); err == nil {
		t.Fatal("validateBackupSetSourceBinding(other source) error = nil, want source mismatch")
	}
}

func TestInspectBackupDestinationClassifiesConfirmedPendingCandidateAndForeignEntries(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	for _, directory := range []string{source, destination, applicationData} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	confirmedIdentity := testArchiveIdentity(storeSourcePath(t, source), store.DestinationPath(), inventory.Digest)
	confirmedIdentity.ArchiveID = "confirmed"
	confirmedIdentity.ConfirmationOrder = 1
	confirmedName := "confirmed.zip"
	confirmedPath := filepath.Join(destination, confirmedName)
	if err := writeBackupArchive(confirmedPath, source, inventory, confirmedIdentity); err != nil {
		t.Fatalf("writeBackupArchive(confirmed) error = %v", err)
	}
	confirmedDigest, err := calculateBackupArchiveDigest(confirmedPath)
	if err != nil {
		t.Fatalf("calculate confirmed digest error = %v", err)
	}
	state := BackupSetState{SchemaVersion: 1, DestinationPath: store.DestinationPath(), SourcePath: confirmedIdentity.SourcePath, NextConfirmationOrder: 3, Archives: []BackupArchiveRecord{{ArchiveID: "confirmed", FileName: confirmedName, ConfirmationOrder: 1, CreatedAt: confirmedIdentity.CreatedAt, WholeArchiveDigest: confirmedDigest, InventoryDigest: inventory.Digest, Lifecycle: BackupArchiveConfirmed}}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save(state) error = %v", err)
	}
	inspection, err := inspectBackupSetDestination(store, source)
	if err != nil || inspection.Status != BackupDestinationInspectionBound || !inspectionHasEntry(inspection, confirmedName, BackupDestinationEntryConfirmed) {
		t.Fatalf("inspect confirmed = %+v, %v, want bound confirmed entry", inspection, err)
	}

	pendingState := state
	pendingState.Archives = append([]BackupArchiveRecord(nil), state.Archives...)
	pendingState.Archives[0].Lifecycle = BackupArchivePendingRemoval
	pendingState.PendingRemoval = &BackupRemovalState{ArchiveID: "confirmed", FileName: confirmedName, ConfirmationOrder: 1}
	if err := store.Save(pendingState); err != nil {
		t.Fatalf("Save(pending state) error = %v", err)
	}
	inspection, err = inspectBackupSetDestination(store, source)
	if err != nil || !inspectionHasEntry(inspection, confirmedName, BackupDestinationEntryPendingRemoval) {
		t.Fatalf("inspect pending = %+v, %v, want pending entry", inspection, err)
	}

	candidateIdentity := confirmedIdentity
	candidateIdentity.ArchiveID = "candidate"
	candidateIdentity.ConfirmationOrder = 2
	candidatePath := filepath.Join(destination, "candidate.zip")
	if err := writeBackupArchive(candidatePath, source, inventory, candidateIdentity); err != nil {
		t.Fatalf("writeBackupArchive(candidate) error = %v", err)
	}
	foreignPath := filepath.Join(destination, "foreign.txt")
	if err := os.WriteFile(foreignPath, []byte("foreign"), 0o600); err != nil {
		t.Fatalf("WriteFile(foreign) error = %v", err)
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save(state for candidates) error = %v", err)
	}
	inspection, err = inspectBackupSetDestination(store, source)
	if err == nil || inspection.Status != BackupDestinationInspectionForeign || !inspectionHasEntry(inspection, "candidate.zip", BackupDestinationEntryOwnedCandidate) || !inspectionHasEntry(inspection, "foreign.txt", BackupDestinationEntryForeign) {
		t.Fatalf("inspect candidates = %+v, %v, want candidate and foreign classification", inspection, err)
	}
	if err := os.Remove(foreignPath); err != nil {
		t.Fatalf("Remove(foreign) error = %v", err)
	}
	if err := recoverOwnedBackupCandidates(store, source); err != nil {
		t.Fatalf("recoverOwnedBackupCandidates() error = %v, want owned candidate cleanup", err)
	}
	if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
		t.Fatalf("candidate stat error = %v, want owned candidate removed", err)
	}
}

func TestInspectBackupDestinationRejectsDamagedHistoryAndRecoveryRemovesOnlyOwnedCandidates(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	for _, directory := range []string{source, destination, applicationData} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	identity := testArchiveIdentity(storeSourcePath(t, source), store.DestinationPath(), inventory.Digest)
	identity.ArchiveID = "confirmed"
	path := filepath.Join(destination, "confirmed.zip")
	if err := writeBackupArchive(path, source, inventory, identity); err != nil {
		t.Fatalf("writeBackupArchive() error = %v", err)
	}
	digest, err := calculateBackupArchiveDigest(path)
	if err != nil {
		t.Fatalf("calculate digest error = %v", err)
	}
	state := BackupSetState{SchemaVersion: 1, DestinationPath: store.DestinationPath(), SourcePath: identity.SourcePath, NextConfirmationOrder: 2, Archives: []BackupArchiveRecord{{ArchiveID: "confirmed", FileName: "confirmed.zip", ConfirmationOrder: 1, CreatedAt: identity.CreatedAt, WholeArchiveDigest: digest, InventoryDigest: inventory.Digest, Lifecycle: BackupArchiveConfirmed}}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save(state) error = %v", err)
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatalf("WriteFile(corrupt) error = %v", err)
	}
	inspection, err := inspectBackupSetDestination(store, source)
	if err == nil || inspection.Status != BackupDestinationInspectionDamaged {
		t.Fatalf("inspect damaged = %+v, %v, want damaged history rejection", inspection, err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove(corrupt) error = %v", err)
	}
	candidateIdentity := identity
	candidateIdentity.ArchiveID = "candidate"
	candidateIdentity.ConfirmationOrder = 2
	candidatePath := filepath.Join(destination, "candidate.zip")
	if err := writeBackupArchive(candidatePath, source, inventory, candidateIdentity); err != nil {
		t.Fatalf("writeBackupArchive(candidate) error = %v", err)
	}
	if err := recoverOwnedBackupCandidates(store, source); err == nil {
		t.Fatal("recoverOwnedBackupCandidates() error = nil, want missing confirmed history rejection")
	}
	if _, err := os.Stat(candidatePath); err != nil {
		t.Fatalf("candidate stat error = %v, want candidate preserved when history is damaged", err)
	}
}

func TestBackupPublicationConfirmationStabilityAndRollback(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	for _, directory := range []string{source, destination, applicationData} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("Mkdir(%q) error = %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("build inventory error = %v", err)
	}
	if err := verifyBackupSourceStability(inventory, inventory, inventory); err != nil {
		t.Fatalf("verifyBackupSourceStability(equal) error = %v", err)
	}
	changed := inventory
	changed.Digest = strings.Repeat("e", 64)
	if err := verifyBackupSourceStability(inventory, changed, inventory); err == nil {
		t.Fatal("verifyBackupSourceStability(changed) error = nil, want instability")
	}

	identity := testArchiveIdentity(storeSourcePath(t, source), store.DestinationPath(), inventory.Digest)
	identity.ArchiveID = "archive-1"
	candidate, err := createBackupArchiveCandidate(destination, identity.ArchiveID)
	if err != nil {
		t.Fatalf("createBackupArchiveCandidate() error = %v", err)
	}
	if filepath.Dir(candidate) != destination || !strings.Contains(filepath.Base(candidate), identity.ArchiveID) {
		t.Fatalf("candidate path = %q, want identifiable destination candidate", candidate)
	}
	if err := writeBackupArchive(candidate, source, inventory, identity); err != nil {
		t.Fatalf("writeBackupArchive(candidate) error = %v", err)
	}
	finalPath := filepath.Join(destination, "archive-1.zip")
	if err := publishBackupArchive(candidate, finalPath); err != nil {
		t.Fatalf("publishBackupArchive() error = %v", err)
	}
	if _, err := os.Stat(candidate); !os.IsNotExist(err) {
		t.Fatalf("candidate stat error = %v, want candidate renamed", err)
	}
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("final archive stat error = %v, want published archive", err)
	}
	digest, err := calculateBackupArchiveDigest(finalPath)
	if err != nil {
		t.Fatalf("calculate final digest error = %v", err)
	}
	state := BackupSetState{SchemaVersion: 1, DestinationPath: store.DestinationPath(), SourcePath: "", NextConfirmationOrder: 1, Archives: []BackupArchiveRecord{}}
	confirmedState, err := confirmBackupArchive(store, state, identity, finalPath, digest)
	if err != nil {
		t.Fatalf("confirmBackupArchive() error = %v", err)
	}
	if confirmedState.SourcePath == "" || len(confirmedState.Archives) != 1 || confirmedState.Archives[0].Lifecycle != BackupArchiveConfirmed {
		t.Fatalf("confirmed state = %+v, want bound confirmed archive", confirmedState)
	}

	firstDestination := filepath.Join(root, "first-destination")
	if err := os.Mkdir(firstDestination, 0o700); err != nil {
		t.Fatalf("Mkdir(first destination) error = %v", err)
	}
	rollbackCandidate := filepath.Join(firstDestination, ".candidate.zip")
	if err := os.WriteFile(rollbackCandidate, []byte("partial"), 0o600); err != nil {
		t.Fatalf("WriteFile(rollback candidate) error = %v", err)
	}
	if err := rollbackBackupFirstCycle(firstDestination, rollbackCandidate); err != nil {
		t.Fatalf("rollbackBackupFirstCycle() error = %v", err)
	}
	if _, err := os.Stat(firstDestination); !os.IsNotExist(err) {
		t.Fatalf("first destination stat error = %v, want rollback removal", err)
	}
}

func TestBackupRetentionPlansMarksResumesAndBlocksAfterDeletionFailure(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	applicationData := filepath.Join(root, "application-data")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	if err := os.Mkdir(applicationData, 0o700); err != nil {
		t.Fatalf("Mkdir(application data) error = %v", err)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}
	digest := strings.Repeat("f", 64)
	state := BackupSetState{SchemaVersion: 1, DestinationPath: store.DestinationPath(), SourcePath: "/source", NextConfirmationOrder: 4}
	for order := uint64(1); order <= 3; order++ {
		name := "archive-" + string(rune('0'+order)) + ".zip"
		if err := os.WriteFile(filepath.Join(destination, name), []byte(name), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
		state.Archives = append(state.Archives, BackupArchiveRecord{ArchiveID: name, FileName: name, ConfirmationOrder: order, CreatedAt: time.Unix(int64(order), 0).UTC(), WholeArchiveDigest: digest, InventoryDigest: digest, Lifecycle: BackupArchiveConfirmed})
	}
	expired, err := planBackupRetention(state, 1)
	if err != nil || len(expired) != 2 || expired[0].ConfirmationOrder != 1 || expired[1].ConfirmationOrder != 2 {
		t.Fatalf("planBackupRetention() = %+v, %v, want oldest two records", expired, err)
	}
	marked, err := markBackupArchivePendingRemoval(store, state, expired[0])
	if err != nil {
		t.Fatalf("markBackupArchivePendingRemoval() error = %v", err)
	}
	if marked.PendingRemoval == nil || marked.Archives[0].Lifecycle != BackupArchivePendingRemoval {
		t.Fatalf("marked state = %+v, want pending removal", marked)
	}
	resumed, err := resumeBackupPendingRemoval(store, marked)
	if err != nil {
		t.Fatalf("resumeBackupPendingRemoval() error = %v", err)
	}
	if resumed.PendingRemoval != nil || len(resumed.Archives) != 2 {
		t.Fatalf("resumed state = %+v, want removed pending record", resumed)
	}

	failureState := state
	failureState.Archives = append([]BackupArchiveRecord(nil), state.Archives...)
	failureArchive := filepath.Join(destination, "archive-1.zip")
	if err := os.Mkdir(failureArchive, 0o700); err != nil {
		t.Fatalf("Mkdir(failure archive) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(failureArchive, "child"), []byte("locked"), 0o600); err != nil {
		t.Fatalf("WriteFile(failure child) error = %v", err)
	}
	if err := executeBackupRetention(store, failureState, 1); err == nil {
		t.Fatal("executeBackupRetention() error = nil, want deletion failure")
	}
	loaded, err := store.Load()
	if err != nil || loaded.State.PendingRemoval == nil {
		t.Fatalf("loaded failed-retention state = %+v, %v, want pending removal", loaded, err)
	}
	if err := validateBackupRetentionCompliance(loaded.State, 1); err == nil {
		t.Fatal("validateBackupRetentionCompliance() error = nil, want publication block")
	}
}

func inspectionHasEntry(inspection BackupDestinationInspection, name string, kind BackupDestinationEntryKind) bool {
	for _, entry := range inspection.Entries {
		if entry.Name == name && entry.Kind == kind {
			return true
		}
	}
	return false
}

func storeSourcePath(t *testing.T, source string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatalf("EvalSymlinks(source) error = %v", err)
	}
	return canonical
}
