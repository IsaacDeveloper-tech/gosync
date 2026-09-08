//go:build !windows

package main

func isLockedFileError(error) bool {
	return false
}
