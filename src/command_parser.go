package gosync

import "fmt"

const watchUsage = "usage: gosync watch <local-directory> <local-directory|ftp://server/remote-directory|sftp://server/remote-directory>"
const configureUsage = "usage: gosync configure"

func parseConfigureCommand(arguments []string) error {
	if len(arguments) != 1 || arguments[0] != "configure" {
		return fmt.Errorf("%s", configureUsage)
	}
	return nil
}

func parseWatchCommand(arguments []string) (RootPaths, error) {
	request, err := parseWatchCommandRequest(arguments)
	if err != nil {
		return RootPaths{}, err
	}
	roots, found := request.LocalRoots()
	if !found {
		return RootPaths{}, fmt.Errorf("remote destination cannot be represented as local synchronization roots")
	}
	return roots, nil
}

func parseWatchCommandRequest(arguments []string) (WatchCommandRequest, error) {
	if len(arguments) != 3 || arguments[0] != "watch" {
		return WatchCommandRequest{}, fmt.Errorf("%s", watchUsage)
	}

	if looksLikeURLArgument(arguments[1]) {
		return WatchCommandRequest{}, fmt.Errorf("watch source must be a local directory")
	}

	if looksLikeURLArgument(arguments[2]) {
		endpoint, err := parseRemoteEndpoint(arguments[2])
		if err != nil {
			return WatchCommandRequest{}, fmt.Errorf("invalid remote destination: %w", err)
		}
		sourceRoot, err := normalizeRootPath(arguments[1])
		if err != nil {
			return WatchCommandRequest{}, fmt.Errorf("normalize local source directory: %w", err)
		}
		return WatchCommandRequest{
			SourceRoot: sourceRoot,
			Destination: WatchDestination{
				Kind:           WatchDestinationRemote,
				RemoteEndpoint: endpoint,
			},
		}, nil
	}

	roots, err := validateRootPaths(arguments[1], arguments[2])
	if err != nil {
		return WatchCommandRequest{}, fmt.Errorf("%s: %w", watchUsage, err)
	}
	return WatchCommandRequest{
		SourceRoot: roots.First,
		Destination: WatchDestination{
			Kind:      WatchDestinationLocal,
			LocalPath: roots.Second,
		},
	}, nil
}
