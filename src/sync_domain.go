package gosync

import "time"

type EntryKind uint8

const (
	EntryKindFile EntryKind = iota
	EntryKindDirectory
)

type SynchronizationEntry struct {
	RelativePath     string
	Kind             EntryKind
	ContentDigest    string
	ModificationTime time.Time
}

type EntryComparison struct {
	RelativePath string
	First        *SynchronizationEntry
	Second       *SynchronizationEntry
	Confirmed    *SynchronizationEntry
}

type SynchronizationActionKind uint8

const (
	SynchronizationActionCopyToFirst SynchronizationActionKind = iota
	SynchronizationActionCopyToSecond
	SynchronizationActionDeleteFromFirst
	SynchronizationActionDeleteFromSecond
)

type SynchronizationAction struct {
	Kind         SynchronizationActionKind
	RelativePath string
}

type SynchronizationResult struct {
	Completed bool
	Failure   error
}
