package gosync

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const BackupArchiveIdentitySchemaVersion = 1

type BackupArchiveIdentity struct {
	SchemaVersion     int
	ArchiveID         string
	SourcePath        string
	DestinationPath   string
	CreatedAt         time.Time
	ConfirmationOrder uint64
	InventoryDigest   string
}

type backupArchiveIdentityDocument struct {
	SchemaVersion     int       `json:"schemaVersion"`
	ArchiveID         string    `json:"archiveID"`
	SourcePath        string    `json:"sourcePath"`
	DestinationPath   string    `json:"destinationPath"`
	CreatedAt         time.Time `json:"createdAt"`
	ConfirmationOrder uint64    `json:"confirmationOrder"`
	InventoryDigest   string    `json:"inventoryDigest"`
}

func validateBackupArchiveIdentity(identity BackupArchiveIdentity, expectedSource, expectedDestination string, expectedOrder uint64, expectedInventoryDigest string) error {
	if identity.SchemaVersion != BackupArchiveIdentitySchemaVersion {
		return fmt.Errorf("unsupported backup archive identity schema version %d", identity.SchemaVersion)
	}
	if strings.TrimSpace(identity.ArchiveID) == "" || strings.TrimSpace(identity.SourcePath) == "" || strings.TrimSpace(identity.DestinationPath) == "" {
		return fmt.Errorf("backup archive identity requires archive and root paths")
	}
	if identity.CreatedAt.IsZero() || identity.ConfirmationOrder == 0 || !isBackupSHA256Digest(identity.InventoryDigest) {
		return errorsInvalidBackupArchiveIdentityValues()
	}
	if expectedSource != "" && identity.SourcePath != expectedSource {
		return fmt.Errorf("backup archive identity source does not match")
	}
	if expectedDestination != "" && identity.DestinationPath != expectedDestination {
		return fmt.Errorf("backup archive identity destination does not match")
	}
	if expectedOrder != 0 && identity.ConfirmationOrder != expectedOrder {
		return fmt.Errorf("backup archive identity confirmation order does not match")
	}
	if expectedInventoryDigest != "" && identity.InventoryDigest != expectedInventoryDigest {
		return fmt.Errorf("backup archive identity inventory digest does not match")
	}
	return nil
}

func errorsInvalidBackupArchiveIdentityValues() error {
	return fmt.Errorf("backup archive identity contains invalid values")
}

func encodeBackupArchiveIdentity(identity BackupArchiveIdentity) (string, error) {
	if err := validateBackupArchiveIdentity(identity, "", "", 0, ""); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(backupArchiveIdentityDocument{
		SchemaVersion:     identity.SchemaVersion,
		ArchiveID:         identity.ArchiveID,
		SourcePath:        identity.SourcePath,
		DestinationPath:   identity.DestinationPath,
		CreatedAt:         identity.CreatedAt,
		ConfirmationOrder: identity.ConfirmationOrder,
		InventoryDigest:   identity.InventoryDigest,
	})
	if err != nil {
		return "", fmt.Errorf("encode backup archive identity: %w", err)
	}
	return string(encoded), nil
}

func decodeBackupArchiveIdentity(encoded string) (BackupArchiveIdentity, error) {
	values, err := decodeStrictBackupObject([]byte(encoded), map[string]struct{}{
		"schemaVersion": {}, "archiveID": {}, "sourcePath": {}, "destinationPath": {}, "createdAt": {}, "confirmationOrder": {}, "inventoryDigest": {},
	})
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	for _, property := range []string{"schemaVersion", "archiveID", "sourcePath", "destinationPath", "createdAt", "confirmationOrder", "inventoryDigest"} {
		if _, found := values[property]; !found {
			return BackupArchiveIdentity{}, fmt.Errorf("missing backup archive identity property %q", property)
		}
	}
	schemaVersion, err := decodeBackupInteger(values["schemaVersion"], "schemaVersion")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	archiveID, err := decodeBackupString(values["archiveID"], "archiveID")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	sourcePath, err := decodeBackupString(values["sourcePath"], "sourcePath")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	destinationPath, err := decodeBackupString(values["destinationPath"], "destinationPath")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	createdAtText, err := decodeBackupString(values["createdAt"], "createdAt")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return BackupArchiveIdentity{}, fmt.Errorf("backup archive identity creation time: %w", err)
	}
	confirmationOrder, err := decodeBackupUint(values["confirmationOrder"], "confirmationOrder")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	inventoryDigest, err := decodeBackupString(values["inventoryDigest"], "inventoryDigest")
	if err != nil {
		return BackupArchiveIdentity{}, err
	}
	identity := BackupArchiveIdentity{
		SchemaVersion:     schemaVersion,
		ArchiveID:         archiveID,
		SourcePath:        sourcePath,
		DestinationPath:   destinationPath,
		CreatedAt:         createdAt,
		ConfirmationOrder: confirmationOrder,
		InventoryDigest:   inventoryDigest,
	}
	if err := validateBackupArchiveIdentity(identity, "", "", 0, ""); err != nil {
		return BackupArchiveIdentity{}, err
	}
	return identity, nil
}

func generateBackupArchiveName(clock func() time.Time, archiveID string) (string, error) {
	if clock == nil {
		return "", fmt.Errorf("backup archive clock is required")
	}
	if archiveID == "" || strings.ContainsAny(archiveID, `/\\`) || archiveID == "." || archiveID == ".." {
		return "", fmt.Errorf("backup archive identifier is invalid")
	}
	return clock().UTC().Format("20060102T150405Z") + "-" + archiveID + ".zip", nil
}
