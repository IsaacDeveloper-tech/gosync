package gosync

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var backupZipEndOfCentralDirectorySignature = []byte{0x50, 0x4b, 0x05, 0x06}

func writeBackupArchive(path string, sourceRoot string, inventory BackupLogicalInventory, identity BackupArchiveIdentity) error {
	if err := validateBackupArchiveIdentity(identity, "", "", 0, inventory.Digest); err != nil {
		return err
	}
	archiveFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create backup archive: %w", err)
	}
	removeArchive := true
	defer func() {
		if removeArchive {
			_ = os.Remove(path)
		}
	}()
	archiveWriter := zip.NewWriter(archiveFile)
	entries := append([]BackupInventoryEntry(nil), inventory.Entries...)
	sort.Slice(entries, func(first, second int) bool { return entries[first].RelativePath < entries[second].RelativePath })
	for _, entry := range entries {
		entryName := entry.RelativePath
		if entry.Kind == BackupInventoryDirectory {
			entryName += "/"
		}
		if err := validateBackupArchiveEntryName(entryName, entry.Kind == BackupInventoryDirectory); err != nil {
			_ = archiveFile.Close()
			return err
		}
		if entry.Kind == BackupInventoryDirectory {
			header := &zip.FileHeader{Name: entryName, Method: zip.Store}
			header.SetMode(os.ModeDir | 0o700)
			if _, err := archiveWriter.CreateHeader(header); err != nil {
				_ = archiveFile.Close()
				return fmt.Errorf("create backup archive directory entry %q: %w", entry.RelativePath, err)
			}
			continue
		}
		if entry.Kind != BackupInventoryFile {
			_ = archiveFile.Close()
			return fmt.Errorf("unsupported backup archive entry kind %q", entry.Kind)
		}
		header := &zip.FileHeader{Name: entry.RelativePath, Method: zip.Deflate}
		header.SetMode(0o600)
		archiveEntry, err := archiveWriter.CreateHeader(header)
		if err != nil {
			_ = archiveFile.Close()
			return fmt.Errorf("create backup archive file entry %q: %w", entry.RelativePath, err)
		}
		sourceFile, err := os.Open(filepath.Join(sourceRoot, filepath.FromSlash(entry.RelativePath)))
		if err != nil {
			_ = archiveFile.Close()
			return fmt.Errorf("open backup source file %q: %w", entry.RelativePath, err)
		}
		_, copyErr := io.Copy(archiveEntry, sourceFile)
		closeErr := sourceFile.Close()
		if copyErr != nil {
			_ = archiveFile.Close()
			return fmt.Errorf("copy backup source file %q: %w", entry.RelativePath, copyErr)
		}
		if closeErr != nil {
			_ = archiveFile.Close()
			return fmt.Errorf("close backup source file %q: %w", entry.RelativePath, closeErr)
		}
	}
	if err := archiveWriter.Close(); err != nil {
		_ = archiveFile.Close()
		return fmt.Errorf("close backup archive writer: %w", err)
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("close backup archive: %w", err)
	}
	identityJSON, err := encodeBackupArchiveIdentity(identity)
	if err != nil {
		return err
	}
	if err := setBackupArchiveComment(path, identityJSON); err != nil {
		return err
	}
	removeArchive = false
	return nil
}

func setBackupArchiveComment(archivePath, comment string) error {
	contents, err := os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("read backup archive for identity: %w", err)
	}
	endOffset := bytes.LastIndex(contents, backupZipEndOfCentralDirectorySignature)
	if endOffset < 0 || endOffset+22 > len(contents) {
		return errors.New("backup archive end record is missing")
	}
	commentBytes := []byte(comment)
	if len(commentBytes) > 65535 {
		return errors.New("backup archive identity is too large")
	}
	oldCommentLength := int(binary.LittleEndian.Uint16(contents[endOffset+20 : endOffset+22]))
	oldEnd := endOffset + 22 + oldCommentLength
	if oldEnd > len(contents) {
		return errors.New("backup archive comment is invalid")
	}
	rewritten := make([]byte, 0, endOffset+22+len(commentBytes))
	rewritten = append(rewritten, contents[:endOffset+22]...)
	binary.LittleEndian.PutUint16(rewritten[endOffset+20:endOffset+22], uint16(len(commentBytes)))
	rewritten = append(rewritten, commentBytes...)
	if err := os.WriteFile(archivePath, rewritten, 0o600); err != nil {
		return fmt.Errorf("write backup archive identity: %w", err)
	}
	return nil
}

func validateBackupArchiveEntryName(name string, directory bool) error {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || filepath.IsAbs(filepath.FromSlash(name)) {
		return fmt.Errorf("unsafe backup archive entry name %q", name)
	}
	withoutSlash := strings.TrimSuffix(name, "/")
	if directory != strings.HasSuffix(name, "/") || withoutSlash == "" {
		return fmt.Errorf("invalid backup archive entry type for %q", name)
	}
	cleanName := path.Clean(withoutSlash)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, "../") || cleanName != withoutSlash {
		return fmt.Errorf("ambiguous backup archive entry name %q", name)
	}
	return nil
}

func verifyBackupArchive(archivePath string, expectedIdentity BackupArchiveIdentity, expectedInventory BackupLogicalInventory) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open backup archive: %w", err)
	}
	defer reader.Close()
	identity, err := decodeBackupArchiveIdentity(reader.Comment)
	if err != nil {
		return fmt.Errorf("decode backup archive identity: %w", err)
	}
	if err := validateBackupArchiveIdentity(identity, expectedIdentity.SourcePath, expectedIdentity.DestinationPath, expectedIdentity.ConfirmationOrder, expectedIdentity.InventoryDigest); err != nil {
		return err
	}
	if identity.ArchiveID != expectedIdentity.ArchiveID || identity.CreatedAt != expectedIdentity.CreatedAt {
		return errors.New("backup archive identity does not match expected archive")
	}
	entries := make([]BackupInventoryEntry, 0, len(reader.File))
	seen := make(map[string]struct{}, len(reader.File))
	for _, file := range reader.File {
		isDirectory := strings.HasSuffix(file.Name, "/")
		if err := validateBackupArchiveEntryName(file.Name, isDirectory); err != nil {
			return err
		}
		if _, found := seen[file.Name]; found {
			return fmt.Errorf("duplicate backup archive entry %q", file.Name)
		}
		seen[file.Name] = struct{}{}
		fileInfo := file.FileInfo()
		entry := BackupInventoryEntry{RelativePath: strings.TrimSuffix(file.Name, "/")}
		switch {
		case isDirectory && fileInfo.IsDir():
			entry.Kind = BackupInventoryDirectory
		case !isDirectory && fileInfo.Mode().IsRegular():
			entry.Kind = BackupInventoryFile
			archiveEntry, err := file.Open()
			if err != nil {
				return fmt.Errorf("open backup archive entry %q: %w", file.Name, err)
			}
			digest := sha256.New()
			_, copyErr := io.Copy(digest, archiveEntry)
			closeErr := archiveEntry.Close()
			if copyErr != nil {
				return fmt.Errorf("read backup archive entry %q: %w", file.Name, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close backup archive entry %q: %w", file.Name, closeErr)
			}
			entry.ContentDigest = hex.EncodeToString(digest.Sum(nil))
		default:
			return fmt.Errorf("unsupported backup archive entry %q", file.Name)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(first, second int) bool { return entries[first].RelativePath < entries[second].RelativePath })
	actualInventory := BackupLogicalInventory{Entries: entries, Digest: backupLogicalInventoryDigest(entries)}
	if !backupLogicalInventoriesEqual(actualInventory, expectedInventory) {
		return errors.New("backup archive logical inventory does not match expected source")
	}
	return nil
}

func calculateBackupArchiveDigest(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open backup archive for digest: %w", err)
	}
	digest := sha256.New()
	_, copyErr := io.Copy(digest, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", fmt.Errorf("read backup archive for digest: %w", copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close backup archive for digest: %w", closeErr)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func verifyBackupArchiveDigest(archivePath, expectedDigest string) error {
	if !isBackupSHA256Digest(expectedDigest) {
		return errors.New("expected backup archive digest is invalid")
	}
	actualDigest, err := calculateBackupArchiveDigest(archivePath)
	if err != nil {
		return err
	}
	if actualDigest != expectedDigest {
		return errors.New("backup archive digest does not match expected digest")
	}
	return nil
}
