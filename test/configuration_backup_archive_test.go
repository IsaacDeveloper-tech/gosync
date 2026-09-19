package gosync_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupArchiveIdentityCodecRoundTripsExactZIPLevelIdentity(t *testing.T) {
	digest := strings.Repeat("a", 64)
	identity := BackupArchiveIdentity{
		SchemaVersion:     BackupArchiveIdentitySchemaVersion,
		ArchiveID:         "archive-1",
		SourcePath:        "/source",
		DestinationPath:   "/destination",
		CreatedAt:         time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		ConfirmationOrder: 4,
		InventoryDigest:   digest,
	}
	encoded, err := encodeBackupArchiveIdentity(identity)
	if err != nil {
		t.Fatalf("encodeBackupArchiveIdentity() error = %v", err)
	}
	want := `{"schemaVersion":1,"archiveID":"archive-1","sourcePath":"/source","destinationPath":"/destination","createdAt":"2026-01-02T03:04:05Z","confirmationOrder":4,"inventoryDigest":"` + digest + `"}`
	if encoded != want {
		t.Fatalf("encoded identity = %q, want %q", encoded, want)
	}
	decoded, err := decodeBackupArchiveIdentity(encoded)
	if err != nil {
		t.Fatalf("decodeBackupArchiveIdentity() error = %v", err)
	}
	if decoded != identity {
		t.Fatalf("decoded identity = %+v, want %+v", decoded, identity)
	}
	if err := validateBackupArchiveIdentity(identity, identity.SourcePath, identity.DestinationPath, identity.ConfirmationOrder, identity.InventoryDigest); err != nil {
		t.Fatalf("validateBackupArchiveIdentity() error = %v", err)
	}
}

func TestBackupArchiveIdentityRejectsInvalidAndForeignValues(t *testing.T) {
	digest := strings.Repeat("b", 64)
	valid := `{"schemaVersion":1,"archiveID":"archive-1","sourcePath":"/source","destinationPath":"/destination","createdAt":"2026-01-02T03:04:05Z","confirmationOrder":4,"inventoryDigest":"` + digest + `"}`
	for _, testCase := range []struct {
		name string
		data string
	}{
		{name: "malformed", data: valid[:len(valid)-1]},
		{name: "duplicate", data: strings.Replace(valid, `"archiveID":"archive-1"`, `"archiveID":"archive-1","archiveID":"other"`, 1)},
		{name: "unknown", data: strings.Replace(valid, `"inventoryDigest":"`+digest+`"`, `"inventoryDigest":"`+digest+`","extra":true`, 1)},
		{name: "missing", data: strings.Replace(valid, `"sourcePath":"/source",`, "", 1)},
		{name: "wrong capitalization", data: strings.Replace(valid, `"sourcePath"`, `"SourcePath"`, 1)},
		{name: "unsupported version", data: strings.Replace(valid, `"schemaVersion":1`, `"schemaVersion":2`, 1)},
		{name: "zero order", data: strings.Replace(valid, `"confirmationOrder":4`, `"confirmationOrder":0`, 1)},
		{name: "bad digest", data: strings.Replace(valid, digest, "not-a-digest", 1)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := decodeBackupArchiveIdentity(testCase.data); err == nil {
				t.Fatalf("decodeBackupArchiveIdentity(%s) error = nil, want rejection", testCase.data)
			}
		})
	}

	identity, err := decodeBackupArchiveIdentity(valid)
	if err != nil {
		t.Fatalf("decodeBackupArchiveIdentity(valid) error = %v", err)
	}
	for name, testCase := range map[string]struct {
		source      string
		destination string
		order       uint64
		digest      string
	}{
		"wrong source":      {source: "/other", destination: identity.DestinationPath, order: identity.ConfirmationOrder, digest: identity.InventoryDigest},
		"wrong destination": {source: identity.SourcePath, destination: "/other", order: identity.ConfirmationOrder, digest: identity.InventoryDigest},
		"wrong order":       {source: identity.SourcePath, destination: identity.DestinationPath, order: 5, digest: identity.InventoryDigest},
		"wrong inventory":   {source: identity.SourcePath, destination: identity.DestinationPath, order: identity.ConfirmationOrder, digest: strings.Repeat("c", 64)},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateBackupArchiveIdentity(identity, testCase.source, testCase.destination, testCase.order, testCase.digest); err == nil {
				t.Fatal("validateBackupArchiveIdentity() error = nil, want foreign or mismatched identity rejection")
			}
		})
	}
}

func TestGenerateBackupArchiveNameUsesUTCAndIdentifierForRepeatedClocks(t *testing.T) {
	clock := func() time.Time {
		return time.Date(2026, time.January, 2, 3, 4, 5, 0, time.FixedZone("local", -5*60*60))
	}
	first, err := generateBackupArchiveName(clock, "id-1")
	if err != nil {
		t.Fatalf("generateBackupArchiveName(first) error = %v", err)
	}
	second, err := generateBackupArchiveName(clock, "id-2")
	if err != nil {
		t.Fatalf("generateBackupArchiveName(second) error = %v", err)
	}
	if first != "20260102T080405Z-id-1.zip" || second != "20260102T080405Z-id-2.zip" || first == second {
		t.Fatalf("archive names = %q and %q, want UTC unique names", first, second)
	}
	if _, err := generateBackupArchiveName(clock, "bad/name"); err == nil {
		t.Fatal("generateBackupArchiveName(unsafe identifier) error = nil, want rejection")
	}
}

func TestBackupDestinationOwnershipRejectsConcurrentOwnersAndSharesCanonicalAliases(t *testing.T) {
	root := t.TempDir()
	applicationData := filepath.Join(root, "application-data")
	destination := filepath.Join(root, "destination")
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
	owner, err := acquireBackupDestinationOwnership(store.LockPath())
	if err != nil {
		t.Fatalf("acquireBackupDestinationOwnership(first) error = %v", err)
	}
	second, err := acquireBackupDestinationOwnership(store.LockPath())
	if err == nil {
		_ = second.Release()
		t.Fatal("acquireBackupDestinationOwnership(second) error = nil, want ownership conflict")
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("Release(first) error = %v", err)
	}
	reacquired, err := acquireBackupDestinationOwnership(store.LockPath())
	if err != nil {
		t.Fatalf("acquireBackupDestinationOwnership(after release) error = %v", err)
	}
	if err := reacquired.Release(); err != nil {
		t.Fatalf("Release(reacquired) error = %v", err)
	}

	alias := filepath.Join(root, "destination-alias")
	if err := os.Symlink(destination, alias); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	aliasStore, err := newBackupSetStoreAt(applicationData, alias)
	if err != nil {
		t.Fatalf("newBackupSetStoreAt(alias) error = %v", err)
	}
	owner, err = acquireBackupDestinationOwnership(store.LockPath())
	if err != nil {
		t.Fatalf("acquireBackupDestinationOwnership(alias first) error = %v", err)
	}
	if _, err := acquireBackupDestinationOwnership(aliasStore.LockPath()); err == nil {
		t.Fatal("acquireBackupDestinationOwnership(alias second) error = nil, want same canonical ownership conflict")
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("Release(alias test) error = %v", err)
	}
}

func TestWriteBackupArchiveCreatesEmptyAndStructuredZIPsWithIdentityComment(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	emptyInventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(empty) error = %v", err)
	}
	identity := testArchiveIdentity(source, filepath.Join(root, "destination"), emptyInventory.Digest)
	emptyArchive := filepath.Join(root, "empty.zip")
	if err := writeBackupArchive(emptyArchive, source, emptyInventory, identity); err != nil {
		t.Fatalf("writeBackupArchive(empty) error = %v", err)
	}
	reader, err := zip.OpenReader(emptyArchive)
	if err != nil {
		t.Fatalf("OpenReader(empty) error = %v", err)
	}
	if len(reader.File) != 0 || reader.Comment == "" {
		t.Fatalf("empty archive files=%d comment=%q, want empty entries and identity comment", len(reader.File), reader.Comment)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close(empty reader) error = %v", err)
	}

	if err := os.Mkdir(filepath.Join(source, "nested"), 0o700); err != nil {
		t.Fatalf("Mkdir(nested) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(source, "nested", "empty"), 0o700); err != nil {
		t.Fatalf("Mkdir(empty directory) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(file) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "zero.txt"), nil, 0o600); err != nil {
		t.Fatalf("WriteFile(zero) error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory(non-empty) error = %v", err)
	}
	identity = testArchiveIdentity(source, filepath.Join(root, "destination"), inventory.Digest)
	archivePath := filepath.Join(root, "structured.zip")
	if err := writeBackupArchive(archivePath, source, inventory, identity); err != nil {
		t.Fatalf("writeBackupArchive(non-empty) error = %v", err)
	}
	reader, err = zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("OpenReader(structured) error = %v", err)
	}
	defer reader.Close()
	if len(reader.File) != len(inventory.Entries) {
		t.Fatalf("structured archive entries = %d, want %d", len(reader.File), len(inventory.Entries))
	}
	for index, file := range reader.File {
		if index > 0 && reader.File[index-1].Name >= file.Name {
			t.Fatalf("ZIP entries are not sorted: %q before %q", reader.File[index-1].Name, file.Name)
		}
		if file.Name == "nested/file.txt" && file.Method != zip.Deflate {
			t.Fatalf("file compression method = %d, want DEFLATE", file.Method)
		}
		if file.Name == "nested/empty/" && !strings.HasSuffix(file.Name, "/") {
			t.Fatalf("empty directory entry = %q, want trailing slash", file.Name)
		}
	}
	if err := verifyBackupArchive(archivePath, identity, inventory); err != nil {
		t.Fatalf("verifyBackupArchive() error = %v", err)
	}
}

func TestWriteBackupArchivePropagatesCreationReadAndPathFailures(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	identity := testArchiveIdentity(source, filepath.Join(root, "destination"), inventory.Digest)
	if err := writeBackupArchive(filepath.Join(root, "missing", "archive.zip"), source, inventory, identity); err == nil {
		t.Fatal("writeBackupArchive(missing destination) error = nil, want creation failure")
	}
	missingSource := filepath.Join(root, "missing-source")
	missingInventory := BackupLogicalInventory{Entries: []BackupInventoryEntry{{RelativePath: "missing.txt", Kind: BackupInventoryFile, ContentDigest: strings.Repeat("a", 64)}}, Digest: strings.Repeat("b", 64)}
	if err := writeBackupArchive(filepath.Join(root, "read-failure.zip"), missingSource, missingInventory, identity); err == nil {
		t.Fatal("writeBackupArchive(missing source) error = nil, want read failure")
	}
	if _, err := os.Stat(filepath.Join(root, "read-failure.zip")); !os.IsNotExist(err) {
		t.Fatalf("failed archive stat error = %v, want no archive publication", err)
	}
}

func TestVerifyBackupArchiveRejectsUnsafeDuplicateMalformedChecksumAndInventoryStructures(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "safe.txt"), []byte("safe"), 0o600); err != nil {
		t.Fatalf("WriteFile(safe) error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	identity := testArchiveIdentity(source, filepath.Join(root, "destination"), inventory.Digest)

	unsafeCases := []struct {
		name      string
		nameInZip string
	}{
		{name: "parent traversal", nameInZip: "../escape.txt"},
		{name: "absolute path", nameInZip: "/escape.txt"},
		{name: "backslash path", nameInZip: `nested\\escape.txt`},
	}
	for _, testCase := range unsafeCases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(root, testCase.name+".zip")
			createZIPWithNames(t, path, []string{testCase.nameInZip})
			if err := verifyBackupArchive(path, identity, inventory); err == nil {
				t.Fatal("verifyBackupArchive() error = nil, want unsafe path rejection")
			}
		})
	}

	duplicatePath := filepath.Join(root, "duplicate.zip")
	createZIPWithNames(t, duplicatePath, []string{"safe.txt", "safe.txt"})
	if err := verifyBackupArchive(duplicatePath, identity, inventory); err == nil {
		t.Fatal("verifyBackupArchive(duplicate) error = nil, want duplicate rejection")
	}

	malformedPath := filepath.Join(root, "malformed.zip")
	if err := os.WriteFile(malformedPath, []byte("not a zip"), 0o600); err != nil {
		t.Fatalf("WriteFile(malformed) error = %v", err)
	}
	if err := verifyBackupArchive(malformedPath, identity, inventory); err == nil {
		t.Fatal("verifyBackupArchive(malformed) error = nil, want malformed rejection")
	}

	validPath := filepath.Join(root, "valid.zip")
	if err := writeBackupArchive(validPath, source, inventory, identity); err != nil {
		t.Fatalf("writeBackupArchive(valid) error = %v", err)
	}
	bytesBefore, err := os.ReadFile(validPath)
	if err != nil {
		t.Fatalf("ReadFile(valid) error = %v", err)
	}
	corrupted := append([]byte(nil), bytesBefore...)
	corrupted[len(corrupted)/2] ^= 0xff
	corruptedPath := filepath.Join(root, "corrupted.zip")
	if err := os.WriteFile(corruptedPath, corrupted, 0o600); err != nil {
		t.Fatalf("WriteFile(corrupted) error = %v", err)
	}
	if err := verifyBackupArchive(corruptedPath, identity, inventory); err == nil {
		t.Fatal("verifyBackupArchive(corrupted) error = nil, want checksum or archive rejection")
	}

	wrongInventory := inventory
	wrongInventory.Digest = strings.Repeat("d", 64)
	if err := verifyBackupArchive(validPath, identity, wrongInventory); err == nil {
		t.Fatal("verifyBackupArchive(wrong inventory) error = nil, want inventory mismatch")
	}
}

func TestBackupArchiveDigestDetectsArchiveMutationsAndTrailingBytes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatalf("Mkdir(source) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatalf("WriteFile(file) error = %v", err)
	}
	inventory, err := buildBackupLogicalInventory(source)
	if err != nil {
		t.Fatalf("buildBackupLogicalInventory() error = %v", err)
	}
	identity := testArchiveIdentity(source, filepath.Join(root, "destination"), inventory.Digest)
	archivePath := filepath.Join(root, "archive.zip")
	if err := writeBackupArchive(archivePath, source, inventory, identity); err != nil {
		t.Fatalf("writeBackupArchive() error = %v", err)
	}
	expectedDigest, err := calculateBackupArchiveDigest(archivePath)
	if err != nil {
		t.Fatalf("calculateBackupArchiveDigest() error = %v", err)
	}
	if len(expectedDigest) != sha256.Size*2 || verifyBackupArchiveDigest(archivePath, expectedDigest) != nil {
		t.Fatalf("archive digest = %q, want valid SHA-256 digest", expectedDigest)
	}

	contents, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile(archive) error = %v", err)
	}
	contents = append(contents, []byte("trailing")...)
	if err := os.WriteFile(archivePath, contents, 0o600); err != nil {
		t.Fatalf("WriteFile(trailing) error = %v", err)
	}
	if err := verifyBackupArchiveDigest(archivePath, expectedDigest); err == nil {
		t.Fatal("verifyBackupArchiveDigest(trailing) error = nil, want digest mismatch")
	}
	if _, err := hex.DecodeString(expectedDigest); err != nil {
		t.Fatalf("expected digest = %q is not hex: %v", expectedDigest, err)
	}
}

func testArchiveIdentity(source, destination, inventoryDigest string) BackupArchiveIdentity {
	return BackupArchiveIdentity{
		SchemaVersion:     BackupArchiveIdentitySchemaVersion,
		ArchiveID:         "archive-test",
		SourcePath:        source,
		DestinationPath:   destination,
		CreatedAt:         time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		ConfirmationOrder: 1,
		InventoryDigest:   inventoryDigest,
	}
}

func createZIPWithNames(t *testing.T, path string, names []string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	writer := zip.NewWriter(file)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			_ = file.Close()
			t.Fatalf("Create(%q) ZIP entry error = %v", name, err)
		}
		if _, err := io.Copy(entry, bytes.NewReader([]byte("content"))); err != nil {
			_ = file.Close()
			t.Fatalf("copy ZIP entry %q error = %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatalf("Close ZIP writer error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close ZIP file error = %v", err)
	}
}
