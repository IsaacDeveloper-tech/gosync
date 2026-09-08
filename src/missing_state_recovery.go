package gosync

import "errors"

type SynchronizationSide uint8

const (
	SynchronizationSideFirst SynchronizationSide = iota
	SynchronizationSideSecond
)

type MissingStateRecovery struct {
	RequiresAuthoritativeSide bool
	AuthoritativeSide         SynchronizationSide
}

func recoverFromMissingConfirmedState(
	confirmedStateFound bool,
	firstInventory DirectoryInventory,
	secondInventory DirectoryInventory,
	requestAuthoritativeSide func() (SynchronizationSide, error),
) (MissingStateRecovery, error) {
	if confirmedStateFound || directoryInventoriesHaveEquivalentContents(firstInventory, secondInventory) {
		return MissingStateRecovery{}, nil
	}
	if requestAuthoritativeSide == nil {
		return MissingStateRecovery{}, errors.New("authoritative side request is required")
	}

	authoritativeSide, err := requestAuthoritativeSide()
	if err != nil {
		return MissingStateRecovery{}, err
	}

	return MissingStateRecovery{
		RequiresAuthoritativeSide: true,
		AuthoritativeSide:         authoritativeSide,
	}, nil
}
