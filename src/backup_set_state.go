package gosync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const BackupSetStateSchemaVersion = 1

type BackupArchiveLifecycle string

const (
	BackupArchiveConfirmed      BackupArchiveLifecycle = "confirmed"
	BackupArchivePendingRemoval BackupArchiveLifecycle = "pending-removal"
)

type BackupArchiveRecord struct {
	ArchiveID          string
	FileName           string
	ConfirmationOrder  uint64
	CreatedAt          time.Time
	WholeArchiveDigest string
	InventoryDigest    string
	Lifecycle          BackupArchiveLifecycle
}

type BackupRemovalState struct {
	ArchiveID         string
	FileName          string
	ConfirmationOrder uint64
}

type BackupSetState struct {
	SchemaVersion         int
	DestinationPath       string
	SourcePath            string
	NextConfirmationOrder uint64
	Archives              []BackupArchiveRecord
	PendingRemoval        *BackupRemovalState
}

func validateBackupSetState(state BackupSetState) error {
	if state.SchemaVersion != BackupSetStateSchemaVersion {
		return fmt.Errorf("unsupported backup set state schema version %d", state.SchemaVersion)
	}
	if strings.TrimSpace(state.DestinationPath) == "" || strings.TrimSpace(state.SourcePath) == "" {
		return errors.New("backup set state source and destination paths are required")
	}
	if state.NextConfirmationOrder == 0 {
		return errors.New("backup set state next confirmation order must be positive")
	}

	archiveIDs := make(map[string]struct{}, len(state.Archives))
	fileNames := make(map[string]struct{}, len(state.Archives))
	orders := make(map[uint64]struct{}, len(state.Archives))
	maximumOrder := uint64(0)
	pendingCount := 0
	for index, archive := range state.Archives {
		if strings.TrimSpace(archive.ArchiveID) == "" || strings.TrimSpace(archive.FileName) == "" {
			return fmt.Errorf("backup archive record %d requires an identifier and file name", index)
		}
		if archive.ConfirmationOrder == 0 {
			return fmt.Errorf("backup archive record %q has invalid confirmation order", archive.ArchiveID)
		}
		if index > 0 && state.Archives[index-1].ConfirmationOrder >= archive.ConfirmationOrder {
			return errors.New("backup archive records must be ordered by confirmation order")
		}
		if _, found := archiveIDs[archive.ArchiveID]; found {
			return fmt.Errorf("duplicate backup archive identifier %q", archive.ArchiveID)
		}
		if _, found := fileNames[archive.FileName]; found {
			return fmt.Errorf("duplicate backup archive file name %q", archive.FileName)
		}
		if _, found := orders[archive.ConfirmationOrder]; found {
			return fmt.Errorf("duplicate backup archive confirmation order %d", archive.ConfirmationOrder)
		}
		if archive.CreatedAt.IsZero() {
			return fmt.Errorf("backup archive record %q requires a creation time", archive.ArchiveID)
		}
		if !isBackupSHA256Digest(archive.WholeArchiveDigest) || !isBackupSHA256Digest(archive.InventoryDigest) {
			return fmt.Errorf("backup archive record %q has an invalid digest", archive.ArchiveID)
		}
		if archive.Lifecycle != BackupArchiveConfirmed && archive.Lifecycle != BackupArchivePendingRemoval {
			return fmt.Errorf("backup archive record %q has unsupported lifecycle %q", archive.ArchiveID, archive.Lifecycle)
		}
		if archive.Lifecycle == BackupArchivePendingRemoval {
			pendingCount++
		}
		archiveIDs[archive.ArchiveID] = struct{}{}
		fileNames[archive.FileName] = struct{}{}
		orders[archive.ConfirmationOrder] = struct{}{}
		maximumOrder = archive.ConfirmationOrder
	}
	if state.NextConfirmationOrder <= maximumOrder {
		return errors.New("backup set state next confirmation order must exceed existing records")
	}
	if pendingCount > 1 {
		return errors.New("backup set state cannot have multiple pending removals")
	}
	if state.PendingRemoval == nil {
		if pendingCount != 0 {
			return errors.New("pending archive record requires pending removal state")
		}
		return nil
	}
	if pendingCount != 1 {
		return errors.New("pending removal state requires one pending archive record")
	}
	for _, archive := range state.Archives {
		if archive.Lifecycle == BackupArchivePendingRemoval &&
			archive.ArchiveID == state.PendingRemoval.ArchiveID &&
			archive.FileName == state.PendingRemoval.FileName &&
			archive.ConfirmationOrder == state.PendingRemoval.ConfirmationOrder {
			return nil
		}
	}
	return errors.New("pending removal state does not match a pending archive record")
}

func isBackupSHA256Digest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

type backupArchiveRecordDocument struct {
	ArchiveID          string                 `json:"archiveID"`
	FileName           string                 `json:"fileName"`
	ConfirmationOrder  uint64                 `json:"confirmationOrder"`
	CreatedAt          time.Time              `json:"createdAt"`
	WholeArchiveDigest string                 `json:"wholeArchiveDigest"`
	InventoryDigest    string                 `json:"inventoryDigest"`
	Lifecycle          BackupArchiveLifecycle `json:"lifecycle"`
}

type backupRemovalStateDocument struct {
	ArchiveID         string `json:"archiveID"`
	FileName          string `json:"fileName"`
	ConfirmationOrder uint64 `json:"confirmationOrder"`
}

type backupSetStateDocument struct {
	SchemaVersion         int                           `json:"schemaVersion"`
	DestinationPath       string                        `json:"destinationPath"`
	SourcePath            string                        `json:"sourcePath"`
	NextConfirmationOrder uint64                        `json:"nextConfirmationOrder"`
	Archives              []backupArchiveRecordDocument `json:"archives"`
	PendingRemoval        *backupRemovalStateDocument   `json:"pendingRemoval"`
}

func encodeBackupSetState(state BackupSetState) (string, error) {
	if err := validateBackupSetState(state); err != nil {
		return "", err
	}
	archives := make([]backupArchiveRecordDocument, len(state.Archives))
	for index, archive := range state.Archives {
		archives[index] = backupArchiveRecordDocument{
			ArchiveID:          archive.ArchiveID,
			FileName:           archive.FileName,
			ConfirmationOrder:  archive.ConfirmationOrder,
			CreatedAt:          archive.CreatedAt,
			WholeArchiveDigest: archive.WholeArchiveDigest,
			InventoryDigest:    archive.InventoryDigest,
			Lifecycle:          archive.Lifecycle,
		}
	}
	document := backupSetStateDocument{
		SchemaVersion:         state.SchemaVersion,
		DestinationPath:       state.DestinationPath,
		SourcePath:            state.SourcePath,
		NextConfirmationOrder: state.NextConfirmationOrder,
		Archives:              archives,
		PendingRemoval:        nil,
	}
	if state.PendingRemoval != nil {
		document.PendingRemoval = &backupRemovalStateDocument{
			ArchiveID:         state.PendingRemoval.ArchiveID,
			FileName:          state.PendingRemoval.FileName,
			ConfirmationOrder: state.PendingRemoval.ConfirmationOrder,
		}
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("encode backup set state: %w", err)
	}
	return string(encoded), nil
}

func decodeBackupSetState(encoded string) (BackupSetState, error) {
	values, err := decodeStrictBackupObject([]byte(encoded), map[string]struct{}{
		"schemaVersion": {}, "destinationPath": {}, "sourcePath": {}, "nextConfirmationOrder": {}, "archives": {}, "pendingRemoval": {},
	})
	if err != nil {
		return BackupSetState{}, err
	}
	for _, property := range []string{"schemaVersion", "destinationPath", "sourcePath", "nextConfirmationOrder", "archives", "pendingRemoval"} {
		if _, found := values[property]; !found {
			return BackupSetState{}, fmt.Errorf("missing backup set state property %q", property)
		}
	}
	schemaVersion, err := decodeBackupInteger(values["schemaVersion"], "schemaVersion")
	if err != nil {
		return BackupSetState{}, err
	}
	destinationPath, err := decodeBackupString(values["destinationPath"], "destinationPath")
	if err != nil {
		return BackupSetState{}, err
	}
	sourcePath, err := decodeBackupString(values["sourcePath"], "sourcePath")
	if err != nil {
		return BackupSetState{}, err
	}
	nextOrder, err := decodeBackupUint(values["nextConfirmationOrder"], "nextConfirmationOrder")
	if err != nil {
		return BackupSetState{}, err
	}
	archives, err := decodeBackupArchiveRecords(values["archives"])
	if err != nil {
		return BackupSetState{}, err
	}
	pendingRemoval, err := decodeBackupRemovalState(values["pendingRemoval"])
	if err != nil {
		return BackupSetState{}, err
	}
	state := BackupSetState{
		SchemaVersion:         schemaVersion,
		DestinationPath:       destinationPath,
		SourcePath:            sourcePath,
		NextConfirmationOrder: nextOrder,
		Archives:              archives,
		PendingRemoval:        pendingRemoval,
	}
	if err := validateBackupSetState(state); err != nil {
		return BackupSetState{}, err
	}
	return state, nil
}

func decodeStrictBackupObject(encoded []byte, allowed map[string]struct{}) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	firstToken, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode backup state object: %w", err)
	}
	delimiter, ok := firstToken.(json.Delim)
	if !ok || delimiter != '{' {
		return nil, errors.New("backup state must be a JSON object")
	}
	values := make(map[string]json.RawMessage, len(allowed))
	for decoder.More() {
		propertyToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode backup state property: %w", err)
		}
		propertyName, ok := propertyToken.(string)
		if !ok {
			return nil, errors.New("backup state property name is invalid")
		}
		if _, ok := allowed[propertyName]; !ok {
			return nil, fmt.Errorf("unknown backup state property %q", propertyName)
		}
		if _, duplicate := values[propertyName]; duplicate {
			return nil, fmt.Errorf("duplicate backup state property %q", propertyName)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("decode backup state property %q: %w", propertyName, err)
		}
		values[propertyName] = value
	}
	closingToken, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("close backup state object: %w", err)
	}
	closingDelimiter, ok := closingToken.(json.Delim)
	if !ok || closingDelimiter != '}' {
		return nil, errors.New("backup state must close as a JSON object")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("backup state contains trailing data")
		}
		return nil, fmt.Errorf("decode backup state trailing data: %w", err)
	}
	return values, nil
}

func decodeBackupString(raw json.RawMessage, property string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("backup state property %q must be a string: %w", property, err)
	}
	return value, nil
}

func decodeBackupInteger(raw json.RawMessage, property string) (int, error) {
	value, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("backup state property %q must be an integer: %w", property, err)
	}
	converted := int(value)
	if int64(converted) != value {
		return 0, fmt.Errorf("backup state property %q is outside the supported integer range", property)
	}
	return converted, nil
}

func decodeBackupUint(raw json.RawMessage, property string) (uint64, error) {
	value, err := strconv.ParseUint(string(raw), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("backup state property %q must be an unsigned integer: %w", property, err)
	}
	return value, nil
}

func decodeBackupArchiveRecords(raw json.RawMessage) ([]BackupArchiveRecord, error) {
	var documents []json.RawMessage
	if err := json.Unmarshal(raw, &documents); err != nil {
		return nil, fmt.Errorf("backup state archives must be an array: %w", err)
	}
	archives := make([]BackupArchiveRecord, len(documents))
	for index, document := range documents {
		values, err := decodeStrictBackupObject(document, map[string]struct{}{
			"archiveID": {}, "fileName": {}, "confirmationOrder": {}, "createdAt": {}, "wholeArchiveDigest": {}, "inventoryDigest": {}, "lifecycle": {},
		})
		if err != nil {
			return nil, fmt.Errorf("decode backup archive record %d: %w", index, err)
		}
		for _, property := range []string{"archiveID", "fileName", "confirmationOrder", "createdAt", "wholeArchiveDigest", "inventoryDigest", "lifecycle"} {
			if _, found := values[property]; !found {
				return nil, fmt.Errorf("missing backup archive property %q", property)
			}
		}
		archiveID, err := decodeBackupString(values["archiveID"], "archiveID")
		if err != nil {
			return nil, err
		}
		fileName, err := decodeBackupString(values["fileName"], "fileName")
		if err != nil {
			return nil, err
		}
		order, err := decodeBackupUint(values["confirmationOrder"], "confirmationOrder")
		if err != nil {
			return nil, err
		}
		createdAtText, err := decodeBackupString(values["createdAt"], "createdAt")
		if err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
		if err != nil {
			return nil, fmt.Errorf("backup archive property %q must be RFC3339 time: %w", "createdAt", err)
		}
		wholeDigest, err := decodeBackupString(values["wholeArchiveDigest"], "wholeArchiveDigest")
		if err != nil {
			return nil, err
		}
		inventoryDigest, err := decodeBackupString(values["inventoryDigest"], "inventoryDigest")
		if err != nil {
			return nil, err
		}
		lifecycleText, err := decodeBackupString(values["lifecycle"], "lifecycle")
		if err != nil {
			return nil, err
		}
		archives[index] = BackupArchiveRecord{
			ArchiveID:          archiveID,
			FileName:           fileName,
			ConfirmationOrder:  order,
			CreatedAt:          createdAt,
			WholeArchiveDigest: wholeDigest,
			InventoryDigest:    inventoryDigest,
			Lifecycle:          BackupArchiveLifecycle(lifecycleText),
		}
	}
	return archives, nil
}

func decodeBackupRemovalState(raw json.RawMessage) (*BackupRemovalState, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	values, err := decodeStrictBackupObject(raw, map[string]struct{}{"archiveID": {}, "fileName": {}, "confirmationOrder": {}})
	if err != nil {
		return nil, fmt.Errorf("decode backup pending removal: %w", err)
	}
	for _, property := range []string{"archiveID", "fileName", "confirmationOrder"} {
		if _, found := values[property]; !found {
			return nil, fmt.Errorf("missing backup pending removal property %q", property)
		}
	}
	archiveID, err := decodeBackupString(values["archiveID"], "archiveID")
	if err != nil {
		return nil, err
	}
	fileName, err := decodeBackupString(values["fileName"], "fileName")
	if err != nil {
		return nil, err
	}
	order, err := decodeBackupUint(values["confirmationOrder"], "confirmationOrder")
	if err != nil {
		return nil, err
	}
	return &BackupRemovalState{ArchiveID: archiveID, FileName: fileName, ConfirmationOrder: order}, nil
}

type BackupSetLoadStatus string

const (
	BackupSetLoadMissing     BackupSetLoadStatus = "missing"
	BackupSetLoadValid       BackupSetLoadStatus = "valid"
	BackupSetLoadInvalid     BackupSetLoadStatus = "invalid"
	BackupSetLoadUnsupported BackupSetLoadStatus = "unsupported-entry"
	BackupSetLoadIOFailure   BackupSetLoadStatus = "io-failure"
)

type BackupSetLoadResult struct {
	Status BackupSetLoadStatus
	State  BackupSetState
}

type BackupSetStore struct {
	applicationDataDirectory string
	destinationPath          string
	statePath                string
	lockPath                 string
}

func newBackupSetStoreAt(applicationDataDirectory, destinationRoot string) (BackupSetStore, error) {
	canonicalApplicationData, err := canonicalizeBackupPath(applicationDataDirectory)
	if err != nil {
		return BackupSetStore{}, fmt.Errorf("canonicalize backup application data: %w", err)
	}
	canonicalDestination, err := canonicalizeBackupPath(destinationRoot)
	if err != nil {
		return BackupSetStore{}, fmt.Errorf("canonicalize backup destination for state: %w", err)
	}
	applicationDataInsideDestination, err := rootContains(canonicalDestination, canonicalApplicationData)
	if err != nil {
		return BackupSetStore{}, err
	}
	if applicationDataInsideDestination {
		return BackupSetStore{}, errors.New("backup application data must remain outside the destination")
	}
	destinationDigest := sha256.Sum256([]byte(canonicalDestination))
	key := hex.EncodeToString(destinationDigest[:])
	return BackupSetStore{
		applicationDataDirectory: canonicalApplicationData,
		destinationPath:          canonicalDestination,
		statePath:                filepath.Join(canonicalApplicationData, "backup-set-"+key+".json"),
		lockPath:                 filepath.Join(canonicalApplicationData, "backup-set-"+key+".lock"),
	}, nil
}

func (store BackupSetStore) DestinationPath() string { return store.destinationPath }
func (store BackupSetStore) StatePath() string       { return store.statePath }
func (store BackupSetStore) LockPath() string        { return store.lockPath }

func (store BackupSetStore) Load() (BackupSetLoadResult, error) {
	entryInfo, err := os.Lstat(store.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			parentInfo, parentErr := os.Lstat(filepath.Dir(store.statePath))
			if parentErr == nil && !parentInfo.IsDir() {
				return BackupSetLoadResult{Status: BackupSetLoadIOFailure}, fmt.Errorf("backup set state parent is not a directory: %w", err)
			}
			return BackupSetLoadResult{Status: BackupSetLoadMissing}, nil
		}
		return BackupSetLoadResult{Status: BackupSetLoadIOFailure}, fmt.Errorf("inspect backup set state: %w", err)
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
		return BackupSetLoadResult{Status: BackupSetLoadUnsupported}, fmt.Errorf("backup set state %q must be a regular file", store.statePath)
	}
	encoded, err := os.ReadFile(store.statePath)
	if err != nil {
		return BackupSetLoadResult{Status: BackupSetLoadIOFailure}, fmt.Errorf("read backup set state: %w", err)
	}
	state, err := decodeBackupSetState(string(encoded))
	if err != nil {
		return BackupSetLoadResult{Status: BackupSetLoadInvalid}, fmt.Errorf("decode backup set state: %w", err)
	}
	if state.DestinationPath != store.destinationPath {
		return BackupSetLoadResult{Status: BackupSetLoadInvalid}, errors.New("backup set state destination does not match requested destination")
	}
	return BackupSetLoadResult{Status: BackupSetLoadValid, State: state}, nil
}

func (store BackupSetStore) WriteCandidate(state BackupSetState) (string, error) {
	encoded, err := encodeBackupSetState(state)
	if err != nil {
		return "", fmt.Errorf("encode backup set state candidate: %w", err)
	}
	if err := os.MkdirAll(store.applicationDataDirectory, 0o700); err != nil {
		return "", fmt.Errorf("create backup state directory: %w", err)
	}
	candidate, err := os.CreateTemp(store.applicationDataDirectory, ".backup-set-*")
	if err != nil {
		return "", fmt.Errorf("create backup state candidate: %w", err)
	}
	candidatePath := candidate.Name()
	removeCandidate := true
	defer func() {
		if removeCandidate {
			_ = os.Remove(candidatePath)
		}
	}()
	if err := candidate.Chmod(0o600); err != nil {
		_ = candidate.Close()
		return "", fmt.Errorf("set backup state candidate permissions: %w", err)
	}
	if _, err := io.WriteString(candidate, encoded); err != nil {
		_ = candidate.Close()
		return "", fmt.Errorf("write backup state candidate: %w", err)
	}
	if err := candidate.Close(); err != nil {
		return "", fmt.Errorf("close backup state candidate: %w", err)
	}
	removeCandidate = false
	return candidatePath, nil
}

func (store BackupSetStore) ReplaceCandidate(candidatePath string) error {
	if filepath.Clean(filepath.Dir(candidatePath)) != filepath.Clean(filepath.Dir(store.statePath)) {
		return errors.New("backup state candidate must be beside active state")
	}
	if err := os.Rename(candidatePath, store.statePath); err != nil {
		if _, statErr := os.Stat(store.statePath); statErr != nil {
			return fmt.Errorf("replace backup set state: %w", err)
		}
		if removeErr := os.Remove(store.statePath); removeErr != nil {
			return fmt.Errorf("replace existing backup set state: %w", removeErr)
		}
		if renameErr := os.Rename(candidatePath, store.statePath); renameErr != nil {
			return fmt.Errorf("replace backup set state after removing previous state: %w", renameErr)
		}
	}
	return nil
}

func (store BackupSetStore) Save(state BackupSetState) error {
	candidatePath, err := store.WriteCandidate(state)
	if err != nil {
		return err
	}
	defer os.Remove(candidatePath)
	return store.ReplaceCandidate(candidatePath)
}
