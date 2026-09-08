//go:build !windows

package gosync

func isLockedFileError(error) bool {
	return false
}
