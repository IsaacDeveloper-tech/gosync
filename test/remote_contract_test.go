package gosync_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestScopedRemoteDestinationExercisesTypedBinaryOperations(t *testing.T) {
	data := []byte{0x00, 0xff, 0x80, 0x01}
	filePath, err := newRemoteRelativePath("nested/archive.bin")
	if err != nil {
		t.Fatalf("NewRemoteRelativePath(file) error = %v", err)
	}
	directoryPath, err := newRemoteRelativePath("nested/empty")
	if err != nil {
		t.Fatalf("NewRemoteRelativePath(directory) error = %v", err)
	}
	candidatePath, err := newRemoteRelativePath(".candidate/archive.tmp")
	if err != nil {
		t.Fatalf("NewRemoteRelativePath(candidate) error = %v", err)
	}
	finalPath, err := newRemoteRelativePath("archive.zip")
	if err != nil {
		t.Fatalf("NewRemoteRelativePath(final) error = %v", err)
	}
	linkPath, _ := newRemoteRelativePath("shortcut")
	specialPath, _ := newRemoteRelativePath("device")
	unknownPath, _ := newRemoteRelativePath("unknown-entry")

	operations := &fakeRemoteProtocolOperations{
		inspection: RemoteDirectoryInspection{
			RootExists: true,
			RootKind:   RemoteEntryDirectory,
			Complete:   true,
			Entries: []RemoteEntry{
				{Path: filePath, Kind: RemoteEntryRegular},
				{Path: directoryPath, Kind: RemoteEntryDirectory},
				{Path: linkPath, Kind: RemoteEntryLink},
				{Path: specialPath, Kind: RemoteEntrySpecial},
				{Path: unknownPath, Kind: RemoteEntryUnknown},
			},
		},
		readContents: data,
	}
	destination, err := newScopedRemoteDestination(testRemoteEndpoint(), operations)
	if err != nil {
		t.Fatalf("NewScopedRemoteDestination() error = %v", err)
	}

	inspection, err := destination.Inspect()
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if inspection.RootKind != RemoteEntryDirectory || !inspection.RootExists || !inspection.Complete || len(inspection.Entries) != 5 {
		t.Fatalf("remote inspection = %+v, want complete selected directory with all typed entries", inspection)
	}
	if inspection.Entries[2].Kind != RemoteEntryLink || inspection.Entries[3].Kind != RemoteEntrySpecial || inspection.Entries[4].Kind != RemoteEntryUnknown {
		t.Fatalf("unsupported entry types were coerced: %+v", inspection.Entries)
	}
	if operations.lastInspectedPath != "/selected/remote/backups" {
		t.Fatalf("inspect path = %q, want selected endpoint root", operations.lastInspectedPath)
	}

	readFile, err := destination.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	readContents, readErr := io.ReadAll(readFile)
	closeErr := readFile.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(readContents, data) {
		t.Fatalf("ReadFile() contents/error/close = %v, %v, %v; want exact binary bytes", readContents, readErr, closeErr)
	}
	if err := destination.WriteFile(filePath, bytes.NewReader(data)); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if !bytes.Equal(operations.writtenContents, data) {
		t.Fatalf("written bytes = %v, want %v", operations.writtenContents, data)
	}
	if err := destination.CreateDirectory(directoryPath); err != nil {
		t.Fatalf("CreateDirectory() error = %v", err)
	}
	if err := destination.DeleteFile(filePath); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if err := destination.DeleteDirectory(directoryPath); err != nil {
		t.Fatalf("DeleteDirectory() error = %v", err)
	}
	if err := destination.Publish(candidatePath, finalPath); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if err := destination.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	wantMutations := []remoteOperationCall{
		{name: "write", firstPath: "/selected/remote/backups/nested/archive.bin"},
		{name: "mkdir", firstPath: "/selected/remote/backups/nested/empty"},
		{name: "delete-file", firstPath: "/selected/remote/backups/nested/archive.bin"},
		{name: "delete-directory", firstPath: "/selected/remote/backups/nested/empty"},
		{name: "publish", firstPath: "/selected/remote/backups/.candidate/archive.tmp", secondPath: "/selected/remote/backups/archive.zip"},
	}
	if !reflect.DeepEqual(operations.mutations, wantMutations) {
		t.Fatalf("remote mutations = %+v, want %+v", operations.mutations, wantMutations)
	}
	if operations.closeCalls != 1 {
		t.Fatalf("transport close calls = %d, want 1", operations.closeCalls)
	}
}

func TestScopedRemoteDestinationEnforcesTheSelectedDirectoryBoundary(t *testing.T) {
	operations := &fakeRemoteProtocolOperations{}
	destination, err := newScopedRemoteDestination(testRemoteEndpoint(), operations)
	if err != nil {
		t.Fatalf("NewScopedRemoteDestination() error = %v", err)
	}

	invalidPaths := []string{"", "/outside/file", "../outside", "nested/../../outside", "nested//file", "nested/./file", `nested\file`, "nested/%2foutside", "nested/%252e%252e/outside", "nested/file/"}
	for _, invalidPath := range invalidPaths {
		if _, err := newRemoteRelativePath(invalidPath); err == nil {
			t.Errorf("NewRemoteRelativePath(%q) error = nil, want boundary rejection", invalidPath)
		}
	}
	outsidePath, err := newRemoteRelativePath("../outside")
	if err == nil {
		t.Fatalf("NewRemoteRelativePath(../outside) error = nil")
	}
	if err := destination.WriteFile(outsidePath, strings.NewReader("unsafe")); err == nil {
		t.Fatal("WriteFile(outside path) error = nil, want boundary rejection")
	}
	if err := destination.DeleteDirectory(remoteRootPath()); err == nil {
		t.Fatal("DeleteDirectory(selected root) error = nil, want root protection")
	}
	if err := destination.WriteFile(remoteRootPath(), strings.NewReader("unsafe")); err == nil {
		t.Fatal("WriteFile(selected root) error = nil, want root protection")
	}
	if len(operations.mutations) != 0 {
		t.Fatalf("mutations after rejected paths = %+v, want none", operations.mutations)
	}

	if _, err := newScopedRemoteDestination(RemoteEndpoint{Protocol: RemoteProtocolFTP, Host: "server.example", Port: 21, Path: "/"}, operations); err == nil {
		t.Fatal("NewScopedRemoteDestination(server root) error = nil, want root endpoint rejection")
	}
	rootPath := remoteRootPath()
	if err := destination.CreateDirectory(rootPath); err != nil {
		t.Fatalf("CreateDirectory(selected root) error = %v, want selected-root creation", err)
	}
	if len(operations.mutations) != 1 || operations.mutations[0].firstPath != "/selected/remote/backups" {
		t.Fatalf("root creation mutations = %+v, want only selected root", operations.mutations)
	}
}

func TestScopedRemoteDestinationPreservesRedirectedRootAndIncompleteListingOutcomes(t *testing.T) {
	testCases := []struct {
		name       string
		inspection RemoteDirectoryInspection
	}{
		{name: "redirected root", inspection: RemoteDirectoryInspection{RootExists: true, RootKind: RemoteEntryLink, Complete: true}},
		{name: "unknown root type", inspection: RemoteDirectoryInspection{RootExists: true, RootKind: RemoteEntryUnknown, Complete: true}},
		{name: "incomplete listing", inspection: RemoteDirectoryInspection{RootExists: true, RootKind: RemoteEntryDirectory, Complete: false}},
		{name: "missing root", inspection: RemoteDirectoryInspection{RootExists: false, Complete: true}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			operations := &fakeRemoteProtocolOperations{inspection: testCase.inspection}
			destination, err := newScopedRemoteDestination(testRemoteEndpoint(), operations)
			if err != nil {
				t.Fatalf("NewScopedRemoteDestination() error = %v", err)
			}
			inspection, err := destination.Inspect()
			if err != nil {
				t.Fatalf("Inspect() error = %v, want typed unsafe outcome returned to caller", err)
			}
			if !reflect.DeepEqual(inspection, testCase.inspection) {
				t.Fatalf("Inspect() = %+v, want unchanged typed outcome %+v", inspection, testCase.inspection)
			}
		})
	}
}

func TestRemoteCapabilityRequirementsRefuseIncompleteServerContracts(t *testing.T) {
	completeCapabilities := RemoteCapabilities{
		CompleteTypedListings: true,
		NonRedirectingPaths:   true,
		LosslessNames:         true,
		BinaryTransfers:       true,
		ReadBackVerification:  true,
		ExclusiveOwnership:    true,
		DurableConfirmation:   true,
	}
	if err := requireRemoteCapabilities(completeCapabilities, SynchronizationModeUnidirectional); err != nil {
		t.Fatalf("RequireRemoteCapabilities(unidirectional) error = %v", err)
	}
	if err := requireRemoteCapabilities(completeCapabilities, SynchronizationModeBackup); err != nil {
		t.Fatalf("RequireRemoteCapabilities(backup) error = %v", err)
	}

	missing := RemoteCapabilities{CompleteTypedListings: true, BinaryTransfers: true}
	wantMissingForMirror := []RemoteCapability{
		RemoteCapabilityNonRedirectingPaths,
		RemoteCapabilityLosslessNames,
		RemoteCapabilityReadBackVerification,
	}
	if got := missing.MissingFor(SynchronizationModeUnidirectional); !reflect.DeepEqual(got, wantMissingForMirror) {
		t.Fatalf("missing mirror capabilities = %v, want %v", got, wantMissingForMirror)
	}
	wantMissingForBackup := append(append([]RemoteCapability(nil), wantMissingForMirror...), RemoteCapabilityExclusiveOwnership, RemoteCapabilityDurableConfirmation)
	if got := missing.MissingFor(SynchronizationModeBackup); !reflect.DeepEqual(got, wantMissingForBackup) {
		t.Fatalf("missing backup capabilities = %v, want %v", got, wantMissingForBackup)
	}
	err := requireRemoteCapabilities(missing, SynchronizationModeBackup)
	var capabilityError *RemoteCapabilityError
	if !errors.As(err, &capabilityError) || !reflect.DeepEqual(capabilityError.Missing, wantMissingForBackup) {
		t.Fatalf("backup capability error = %#v, want ordered missing capability refusal", err)
	}
	if err := requireRemoteCapabilities(completeCapabilities, SynchronizationModeBidirectional); err == nil {
		t.Fatal("RequireRemoteCapabilities(bidirectional) error = nil, want unsupported-mode refusal")
	}
}

func TestRemoteContractPreservesPrimaryCleanupAndCloseFailures(t *testing.T) {
	operationError := errors.New("upload failed")
	cleanupError := errors.New("candidate cleanup failed")
	closeError := errors.New("remote session close failed")
	failure := combineRemoteOperationErrors(operationError, cleanupError, closeError)
	if !errors.Is(failure, operationError) || !errors.Is(failure, cleanupError) || !errors.Is(failure, closeError) {
		t.Fatalf("combined error = %v, want all primary, cleanup, and close causes", failure)
	}
	var typedFailure *RemoteOperationFailure
	if !errors.As(failure, &typedFailure) || typedFailure.OperationError != operationError || typedFailure.CleanupError != cleanupError || typedFailure.CloseError != closeError {
		t.Fatalf("typed remote failure = %#v, want each cause stored independently", typedFailure)
	}
	if combineRemoteOperationErrors(nil, nil, nil) != nil {
		t.Fatal("combineRemoteOperationErrors(no failures) must return nil")
	}
}

func TestRemoteContractFakeCanInjectEveryProtocolOperationFailure(t *testing.T) {
	entryPath, err := newRemoteRelativePath("file.bin")
	if err != nil {
		t.Fatalf("NewRemoteRelativePath() error = %v", err)
	}
	candidatePath, _ := newRemoteRelativePath("candidate.tmp")
	finalPath, _ := newRemoteRelativePath("final.zip")
	operationsToFail := []struct {
		name      string
		invoke    func(RemoteDestination) error
		configure func(*fakeRemoteProtocolOperations, error)
	}{
		{name: "inspect", invoke: func(destination RemoteDestination) error { _, err := destination.Inspect(); return err }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.inspectErr = err }},
		{name: "read", invoke: func(destination RemoteDestination) error { _, err := destination.ReadFile(entryPath); return err }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.readErr = err }},
		{name: "write", invoke: func(destination RemoteDestination) error {
			return destination.WriteFile(entryPath, strings.NewReader("bytes"))
		}, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.writeErr = err }},
		{name: "create directory", invoke: func(destination RemoteDestination) error { return destination.CreateDirectory(entryPath) }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.createDirectoryErr = err }},
		{name: "delete file", invoke: func(destination RemoteDestination) error { return destination.DeleteFile(entryPath) }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.deleteFileErr = err }},
		{name: "delete directory", invoke: func(destination RemoteDestination) error { return destination.DeleteDirectory(entryPath) }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.deleteDirectoryErr = err }},
		{name: "publish", invoke: func(destination RemoteDestination) error { return destination.Publish(candidatePath, finalPath) }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.publishErr = err }},
		{name: "capabilities", invoke: func(destination RemoteDestination) error { _, err := destination.Capabilities(); return err }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.capabilitiesErr = err }},
		{name: "close", invoke: func(destination RemoteDestination) error { return destination.Close() }, configure: func(fake *fakeRemoteProtocolOperations, err error) { fake.closeErr = err }},
	}
	for _, operation := range operationsToFail {
		t.Run(operation.name, func(t *testing.T) {
			failure := errors.New("injected no-network failure")
			fake := &fakeRemoteProtocolOperations{}
			operation.configure(fake, failure)
			destination, err := newScopedRemoteDestination(testRemoteEndpoint(), fake)
			if err != nil {
				t.Fatalf("NewScopedRemoteDestination() error = %v", err)
			}
			if err := operation.invoke(destination); !errors.Is(err, failure) {
				t.Fatalf("remote operation error = %v, want injected cause", err)
			}
		})
	}
}

func testRemoteEndpoint() RemoteEndpoint {
	return RemoteEndpoint{Protocol: RemoteProtocolFTP, Host: "server.example", Port: 21, Path: "/selected/remote/backups"}
}

type remoteOperationCall struct {
	name       string
	firstPath  string
	secondPath string
}

type fakeRemoteProtocolOperations struct {
	inspection         RemoteDirectoryInspection
	readContents       []byte
	writtenContents    []byte
	lastInspectedPath  string
	mutations          []remoteOperationCall
	capabilities       RemoteCapabilities
	inspectErr         error
	readErr            error
	writeErr           error
	createDirectoryErr error
	deleteFileErr      error
	deleteDirectoryErr error
	publishErr         error
	capabilitiesErr    error
	closeErr           error
	closeCalls         int
}

func (operations *fakeRemoteProtocolOperations) InspectDirectory(path string) (RemoteDirectoryInspection, error) {
	operations.lastInspectedPath = path
	return operations.inspection, operations.inspectErr
}

func (operations *fakeRemoteProtocolOperations) ReadFile(string) (io.ReadCloser, error) {
	if operations.readErr != nil {
		return nil, operations.readErr
	}
	return io.NopCloser(bytes.NewReader(operations.readContents)), nil
}

func (operations *fakeRemoteProtocolOperations) WriteFile(path string, contents io.Reader) error {
	if operations.writeErr != nil {
		return operations.writeErr
	}
	writtenContents, err := io.ReadAll(contents)
	if err != nil {
		return err
	}
	operations.writtenContents = writtenContents
	operations.mutations = append(operations.mutations, remoteOperationCall{name: "write", firstPath: path})
	return nil
}

func (operations *fakeRemoteProtocolOperations) CreateDirectory(path string) error {
	if operations.createDirectoryErr != nil {
		return operations.createDirectoryErr
	}
	operations.mutations = append(operations.mutations, remoteOperationCall{name: "mkdir", firstPath: path})
	return nil
}

func (operations *fakeRemoteProtocolOperations) DeleteFile(path string) error {
	if operations.deleteFileErr != nil {
		return operations.deleteFileErr
	}
	operations.mutations = append(operations.mutations, remoteOperationCall{name: "delete-file", firstPath: path})
	return nil
}

func (operations *fakeRemoteProtocolOperations) DeleteDirectory(path string) error {
	if operations.deleteDirectoryErr != nil {
		return operations.deleteDirectoryErr
	}
	operations.mutations = append(operations.mutations, remoteOperationCall{name: "delete-directory", firstPath: path})
	return nil
}

func (operations *fakeRemoteProtocolOperations) Publish(candidatePath, finalPath string) error {
	if operations.publishErr != nil {
		return operations.publishErr
	}
	operations.mutations = append(operations.mutations, remoteOperationCall{name: "publish", firstPath: candidatePath, secondPath: finalPath})
	return nil
}

func (operations *fakeRemoteProtocolOperations) Capabilities() (RemoteCapabilities, error) {
	return operations.capabilities, operations.capabilitiesErr
}

func (operations *fakeRemoteProtocolOperations) Close() error {
	operations.closeCalls++
	return operations.closeErr
}
