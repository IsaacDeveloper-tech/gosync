package gosync_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildBackupLogicalInventoryRepresentsCanonicalEntriesAndEmptySources(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatalf("Mkdir(nested) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested", "empty"), 0o700); err != nil {
		t.Fatalf("Mkdir(empty) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(file) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "zero.txt"), nil, 0o600); err != nil {
		t.Fatalf("WriteFile(zero) error = %v", err)
	}

	inventory, err := buildBackupLogicalInventory(root)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	if len(inventory.Entries) != 4 {
		t.Fatalf("inventory entries = %d, want 4", len(inventory.Entries))
	}
	if !backupLogicalInventoriesEqual(inventory, inventory) {
		t.Fatal("inventory is not equivalent to itself")
	}
	for index := 1; index < len(inventory.Entries); index++ {
		if inventory.Entries[index-1].RelativePath >= inventory.Entries[index].RelativePath {
			t.Fatalf("inventory entries are not deterministically sorted: %+v", inventory.Entries)
		}
	}

	contentDigest := sha256.Sum256([]byte("content"))
	emptyDigest := sha256.Sum256(nil)
	entries := map[string]BackupInventoryEntry{}
	for _, entry := range inventory.Entries {
		entries[entry.RelativePath] = entry
	}
	if entries["nested"].Kind != BackupInventoryDirectory || entries["nested/empty"].Kind != BackupInventoryDirectory {
		t.Fatalf("directory entries = %+v, want directory kinds", entries)
	}
	if entries["nested/file.txt"].Kind != BackupInventoryFile || entries["nested/file.txt"].ContentDigest != hex.EncodeToString(contentDigest[:]) {
		t.Fatalf("file entry = %+v, want content digest", entries["nested/file.txt"])
	}
	if entries["zero.txt"].Kind != BackupInventoryFile || entries["zero.txt"].ContentDigest != hex.EncodeToString(emptyDigest[:]) {
		t.Fatalf("zero file entry = %+v, want empty-file digest", entries["zero.txt"])
	}

	emptyInventory, err := buildBackupLogicalInventory(t.TempDir())
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(empty) error = %v", err)
	}
	if len(emptyInventory.Entries) != 0 || emptyInventory.Digest == "" {
		t.Fatalf("empty inventory = %+v, want empty entries and aggregate digest", emptyInventory)
	}
}

func TestBackupLogicalInventoryDigestIsIndependentOfTraversalAndExcludedMetadata(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	for _, root := range []string{firstRoot, secondRoot} {
		if err := os.Mkdir(filepath.Join(root, "directory"), 0o700); err != nil {
			t.Fatalf("Mkdir(directory) error = %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "directory", "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatalf("WriteFile(first a) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "directory", "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatalf("WriteFile(first b) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "directory", "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatalf("WriteFile(second b) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "directory", "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatalf("WriteFile(second a) error = %v", err)
	}

	first, err := buildBackupLogicalInventory(firstRoot)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(first) error = %v", err)
	}
	second, err := buildBackupLogicalInventory(secondRoot)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(second) error = %v", err)
	}
	if first.Digest != second.Digest || !backupLogicalInventoriesEqual(first, second) {
		t.Fatalf("equal logical trees differ: first=%+v second=%+v", first, second)
	}

	oldTime := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(firstRoot, "directory", "a.txt"), oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(first) error = %v", err)
	}
	if err := os.Chtimes(filepath.Join(secondRoot, "directory", "a.txt"), newTime, newTime); err != nil {
		t.Fatalf("Chtimes(second) error = %v", err)
	}
	first, err = buildBackupLogicalInventory(firstRoot)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(first metadata) error = %v", err)
	}
	second, err = buildBackupLogicalInventory(secondRoot)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(second metadata) error = %v", err)
	}
	if first.Digest != second.Digest || !backupLogicalInventoriesEqual(first, second) {
		t.Fatal("metadata-only changes changed logical inventory")
	}

	if err := os.WriteFile(filepath.Join(secondRoot, "directory", "a.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatalf("WriteFile(changed) error = %v", err)
	}
	second, err = buildBackupLogicalInventory(secondRoot)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(changed) error = %v", err)
	}
	if first.Digest == second.Digest || backupLogicalInventoriesEqual(first, second) {
		t.Fatal("content change did not change logical inventory")
	}
}

func TestBackupLogicalInventoryRejectsUnsupportedEntriesAndClassifiesBlockedErrors(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, err := buildBackupLogicalInventory(root)
	if err == nil {
		t.Fatal("buildBackupLogicalInventory(symlink) error = nil, want unsupported-entry error")
	}
	var inventoryError *BackupInventoryError
	if !errors.As(err, &inventoryError) || inventoryError.Kind != BackupInventoryErrorUnsupported {
		t.Fatalf("inventory error = %v, want unsupported-entry classification", err)
	}

	blockedError := &BackupInventoryError{Path: "locked.txt", Kind: BackupInventoryErrorBlocked}
	if !isBackupInventoryBlockedError(blockedError) {
		t.Fatal("blocked inventory error was not classified as blocked")
	}
}

func TestBackupSetStateRepresentsRecordsAndPendingRemoval(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	digest := strings.Repeat("a", 64)
	state := BackupSetState{
		SchemaVersion:         BackupSetStateSchemaVersion,
		DestinationPath:       `C:\\backups`,
		SourcePath:            `C:\\source`,
		NextConfirmationOrder: 3,
		Archives: []BackupArchiveRecord{
			{
				ArchiveID:          "archive-1",
				FileName:           "20260102T030405Z-archive-1.zip",
				ConfirmationOrder:  1,
				CreatedAt:          createdAt,
				WholeArchiveDigest: digest,
				InventoryDigest:    digest,
				Lifecycle:          BackupArchiveConfirmed,
			},
			{
				ArchiveID:          "archive-2",
				FileName:           "20260102T030406Z-archive-2.zip",
				ConfirmationOrder:  2,
				CreatedAt:          createdAt.Add(time.Second),
				WholeArchiveDigest: digest,
				InventoryDigest:    digest,
				Lifecycle:          BackupArchivePendingRemoval,
			},
		},
		PendingRemoval: &BackupRemovalState{ArchiveID: "archive-2", FileName: "20260102T030406Z-archive-2.zip", ConfirmationOrder: 2},
	}
	if err := validateBackupSetState(state); err != nil {
		t.Fatalf("validateBackupSetState() error = %v", err)
	}
}

func TestBackupSetStateCodecUsesExactSchemaAndRoundTrips(t *testing.T) {
	digest := strings.Repeat("b", 64)
	state := BackupSetState{
		SchemaVersion:         BackupSetStateSchemaVersion,
		DestinationPath:       "/backups",
		SourcePath:            "/source",
		NextConfirmationOrder: 2,
		Archives: []BackupArchiveRecord{{
			ArchiveID:          "id-1",
			FileName:           "20260102T030405Z-id-1.zip",
			ConfirmationOrder:  1,
			CreatedAt:          time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
			WholeArchiveDigest: digest,
			InventoryDigest:    digest,
			Lifecycle:          BackupArchiveConfirmed,
		}},
	}
	encoded, err := encodeBackupSetState(state)
	if err != nil {
		t.Fatalf("encodeBackupSetState() error = %v", err)
	}
	want := `{"schemaVersion":1,"destinationPath":"/backups","sourcePath":"/source","nextConfirmationOrder":2,"archives":[{"archiveID":"id-1","fileName":"20260102T030405Z-id-1.zip","confirmationOrder":1,"createdAt":"2026-01-02T03:04:05Z","wholeArchiveDigest":"` + digest + `","inventoryDigest":"` + digest + `","lifecycle":"confirmed"}],"pendingRemoval":null}`
	if encoded != want {
		t.Fatalf("encoded state = %q, want %q", encoded, want)
	}
	decoded, err := decodeBackupSetState(encoded)
	if err != nil {
		t.Fatalf("decodeBackupSetState() error = %v", err)
	}
	if len(decoded.Archives) != 1 || decoded.Archives[0].ArchiveID != "id-1" || decoded.PendingRemoval != nil {
		t.Fatalf("decoded state = %+v, want exact state", decoded)
	}
}

func TestDecodeBackupSetStateRejectsAmbiguousAndInvalidDocuments(t *testing.T) {
	digest := strings.Repeat("c", 64)
	valid := `{"schemaVersion":1,"destinationPath":"/backups","sourcePath":"/source","nextConfirmationOrder":2,"archives":[{"archiveID":"id-1","fileName":"one.zip","confirmationOrder":1,"createdAt":"2026-01-02T03:04:05Z","wholeArchiveDigest":"` + digest + `","inventoryDigest":"` + digest + `","lifecycle":"confirmed"}],"pendingRemoval":null}`
	testCases := []struct {
		name string
		data string
	}{
		{name: "malformed", data: valid[:len(valid)-1]},
		{name: "duplicate property", data: strings.Replace(valid, `"sourcePath":"/source"`, `"sourcePath":"/source","sourcePath":"/other"`, 1)},
		{name: "unknown property", data: strings.Replace(valid, `"pendingRemoval":null`, `"pendingRemoval":null,"extra":true`, 1)},
		{name: "missing property", data: strings.Replace(valid, `"sourcePath":"/source",`, "", 1)},
		{name: "wrong capitalization", data: strings.Replace(valid, `"sourcePath"`, `"SourcePath"`, 1)},
		{name: "unsupported state version", data: strings.Replace(valid, `"schemaVersion":1`, `"schemaVersion":2`, 1)},
		{name: "zero next order", data: strings.Replace(valid, `"nextConfirmationOrder":2`, `"nextConfirmationOrder":0`, 1)},
		{name: "zero archive order", data: strings.Replace(valid, `"confirmationOrder":1`, `"confirmationOrder":0`, 1)},
		{name: "invalid lifecycle", data: strings.Replace(valid, `"lifecycle":"confirmed"`, `"lifecycle":"unknown"`, 1)},
		{name: "missing pending marker", data: strings.Replace(valid, `"pendingRemoval":null`, `"pendingRemoval":{}`, 1)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := decodeBackupSetState(testCase.data); err == nil {
				t.Fatalf("decodeBackupSetState(%s) error = nil, want rejection", testCase.data)
			}
		})
	}
}

func TestBackupSetStoreResolvesStableDestinationKeyAndKeepsStateOutsideDestination(t *testing.T) {
	root := t.TempDir()
	applicationData := filepath.Join(root, "application-data")
	destination := filepath.Join(root, "backups")
	if err := os.Mkdir(applicationData, 0o700); err != nil {
		t.Fatalf("Mkdir(application data) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}

	first, err := newBackupSetStoreAt(applicationData, filepath.Join(root, "backups", "."))
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(first) error = %v", err)
	}
	second, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(second) error = %v", err)
	}
	if first.StatePath() != second.StatePath() || first.LockPath() != second.LockPath() {
		t.Fatalf("destination aliases produced different paths: first=%q/%q second=%q/%q", first.StatePath(), first.LockPath(), second.StatePath(), second.LockPath())
	}
	if filepath.IsAbs(first.StatePath()) == false || filepath.IsAbs(first.LockPath()) == false {
		t.Fatal("state and lock paths must be absolute")
	}
	if relative, err := filepath.Rel(destination, first.StatePath()); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("state path %q is inside destination", first.StatePath())
	}
	if relative, err := filepath.Rel(destination, first.LockPath()); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("lock path %q is inside destination", first.LockPath())
	}
}

func TestBackupSetStoreLoadsTypedOutcomesAndAtomicallyReplacesState(t *testing.T) {
	root := t.TempDir()
	applicationData := filepath.Join(root, "application-data")
	destination := filepath.Join(root, "backups")
	if err := os.Mkdir(applicationData, 0o700); err != nil {
		t.Fatalf("Mkdir(application data) error = %v", err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatalf("Mkdir(destination) error = %v", err)
	}
	store, err := newBackupSetStoreAt(applicationData, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt() error = %v", err)
	}

	result, err := store.Load()
	if err != nil || result.Status != BackupSetLoadMissing {
		t.Fatalf("Load(missing) = %+v, %v, want missing without error", result, err)
	}
	digest := strings.Repeat("d", 64)
	canonicalDestination, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatalf("EvalSymlinks(destination) error = %v", err)
	}
	state := BackupSetState{SchemaVersion: 1, DestinationPath: canonicalDestination, SourcePath: filepath.Join(root, "source"), NextConfirmationOrder: 1, Archives: []BackupArchiveRecord{}, PendingRemoval: nil}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save(state) error = %v", err)
	}
	result, err = store.Load()
	if err != nil || result.Status != BackupSetLoadValid || result.State.DestinationPath != canonicalDestination {
		t.Fatalf("Load(valid) = %+v, %v, want valid state", result, err)
	}

	if err := os.WriteFile(store.StatePath(), []byte("invalid state"), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid state) error = %v", err)
	}
	result, err = store.Load()
	if err == nil || result.Status != BackupSetLoadInvalid {
		t.Fatalf("Load(invalid) = %+v, %v, want invalid outcome", result, err)
	}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save(state after invalid) error = %v", err)
	}

	unsupportedPath := filepath.Join(root, "unsupported-state")
	if err := os.WriteFile(unsupportedPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(unsupported state) error = %v", err)
	}
	unsupportedStore, err := newBackupSetStoreAt(unsupportedPath, destination)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(unsupported) error = %v", err)
	}
	result, err = unsupportedStore.Load()
	if err == nil || result.Status != BackupSetLoadIOFailure {
		t.Fatalf("Load(inaccessible state) = %+v, %v, want I/O failure", result, err)
	}

	_ = digest
}
