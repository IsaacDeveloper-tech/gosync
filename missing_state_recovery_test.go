package main

import (
	"errors"
	"testing"
)

func TestRecoverFromMissingConfirmedStateDoesNothingWhenStateExists(t *testing.T) {
	requestCalls := 0
	requestAuthoritativeSide := func() (SynchronizationSide, error) {
		requestCalls++
		return SynchronizationSideFirst, nil
	}

	recovery, err := recoverFromMissingConfirmedState(
		true,
		DirectoryInventory{"notes.txt": {RelativePath: "notes.txt", Kind: EntryKindFile, ContentDigest: "first"}},
		DirectoryInventory{"notes.txt": {RelativePath: "notes.txt", Kind: EntryKindFile, ContentDigest: "second"}},
		requestAuthoritativeSide,
	)
	if err != nil {
		t.Fatalf("recoverFromMissingConfirmedState() error = %v", err)
	}
	if recovery.RequiresAuthoritativeSide {
		t.Fatal("recovery requires an authoritative side, want false when state exists")
	}
	if requestCalls != 0 {
		t.Fatalf("authoritative side request calls = %d, want 0", requestCalls)
	}
}

func TestRecoverFromMissingConfirmedStateDoesNothingWhenDirectoriesMatch(t *testing.T) {
	requestCalls := 0
	requestAuthoritativeSide := func() (SynchronizationSide, error) {
		requestCalls++
		return SynchronizationSideFirst, nil
	}
	inventory := DirectoryInventory{
		"notes.txt": {RelativePath: "notes.txt", Kind: EntryKindFile, ContentDigest: "same"},
	}

	recovery, err := recoverFromMissingConfirmedState(false, inventory, inventory, requestAuthoritativeSide)
	if err != nil {
		t.Fatalf("recoverFromMissingConfirmedState() error = %v", err)
	}
	if recovery.RequiresAuthoritativeSide {
		t.Fatal("recovery requires an authoritative side, want false when directories match")
	}
	if requestCalls != 0 {
		t.Fatalf("authoritative side request calls = %d, want 0", requestCalls)
	}
}

func TestRecoverFromMissingConfirmedStateRequestsAuthoritativeSideWhenDirectoriesDiffer(t *testing.T) {
	requestCalls := 0
	requestAuthoritativeSide := func() (SynchronizationSide, error) {
		requestCalls++
		return SynchronizationSideSecond, nil
	}

	recovery, err := recoverFromMissingConfirmedState(
		false,
		DirectoryInventory{"first.txt": {RelativePath: "first.txt", Kind: EntryKindFile, ContentDigest: "first"}},
		DirectoryInventory{"second.txt": {RelativePath: "second.txt", Kind: EntryKindFile, ContentDigest: "second"}},
		requestAuthoritativeSide,
	)
	if err != nil {
		t.Fatalf("recoverFromMissingConfirmedState() error = %v", err)
	}
	if !recovery.RequiresAuthoritativeSide {
		t.Fatal("recovery requires an authoritative side = false, want true")
	}
	if recovery.AuthoritativeSide != SynchronizationSideSecond {
		t.Fatalf("authoritative side = %v, want %v", recovery.AuthoritativeSide, SynchronizationSideSecond)
	}
	if requestCalls != 1 {
		t.Fatalf("authoritative side request calls = %d, want 1", requestCalls)
	}
}

func TestRecoverFromMissingConfirmedStatePropagatesAuthoritativeSideError(t *testing.T) {
	wantError := errors.New("user cancelled recovery")
	requestAuthoritativeSide := func() (SynchronizationSide, error) {
		return SynchronizationSideFirst, wantError
	}

	_, err := recoverFromMissingConfirmedState(
		false,
		DirectoryInventory{"first.txt": {RelativePath: "first.txt", Kind: EntryKindFile, ContentDigest: "first"}},
		DirectoryInventory{"second.txt": {RelativePath: "second.txt", Kind: EntryKindFile, ContentDigest: "second"}},
		requestAuthoritativeSide,
	)
	if !errors.Is(err, wantError) {
		t.Fatalf("recoverFromMissingConfirmedState() error = %v, want %v", err, wantError)
	}
}
