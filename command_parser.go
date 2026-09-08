package main

import "fmt"

const watchUsage = "usage: gosync watch <directory-a> <directory-b>"

func parseWatchCommand(arguments []string) (RootPaths, error) {
	if len(arguments) != 3 || arguments[0] != "watch" {
		return RootPaths{}, fmt.Errorf("%s", watchUsage)
	}

	roots, err := validateRootPaths(arguments[1], arguments[2])
	if err != nil {
		return RootPaths{}, fmt.Errorf("%s: %w", watchUsage, err)
	}

	return roots, nil
}
