package gosync_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWatchCommandRequestSeparatesLocalAndRemoteDestinations(t *testing.T) {
	temporaryDirectory := t.TempDir()
	localSource := filepath.Join(temporaryDirectory, "source")
	localDestination := filepath.Join(temporaryDirectory, "destination")
	localRequest, err := parseWatchCommandRequest([]string{"watch", localSource, localDestination})
	if err != nil {
		t.Fatalf("parseWatchCommandRequest(local) error = %v", err)
	}
	localRoots, found := localRequest.LocalRoots()
	if !found || localRoots.First != localSource || localRoots.Second != localDestination {
		t.Fatalf("local request roots = %+v, found = %t, want local source and destination", localRoots, found)
	}
	if localRequest.Destination.RemoteEndpoint != (RemoteEndpoint{}) {
		t.Fatalf("local request contains remote endpoint %+v", localRequest.Destination.RemoteEndpoint)
	}

	testCases := []struct {
		name       string
		url        string
		protocol   RemoteProtocol
		port       int
		remotePath string
	}{
		{name: "ftp default port", url: "ftp://Backup.Example/archive", protocol: RemoteProtocolFTP, port: 21, remotePath: "/archive"},
		{name: "sftp custom port and escaped space", url: "sftp://backup.example:2222/weekly%20copies", protocol: RemoteProtocolSFTP, port: 2222, remotePath: "/weekly copies"},
		{name: "case-insensitive scheme", url: "FTP://BACKUP.EXAMPLE:2121/archive", protocol: RemoteProtocolFTP, port: 2121, remotePath: "/archive"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request, err := parseWatchCommandRequest([]string{"watch", localSource, testCase.url})
			if err != nil {
				t.Fatalf("parseWatchCommandRequest() error = %v", err)
			}
			if request.Destination.Kind != WatchDestinationRemote {
				t.Fatalf("destination kind = %q, want remote", request.Destination.Kind)
			}
			if request.Destination.LocalPath != "" {
				t.Fatalf("remote destination has local path %q", request.Destination.LocalPath)
			}
			endpoint := request.Destination.RemoteEndpoint
			if endpoint.Protocol != testCase.protocol || endpoint.Host != "backup.example" || endpoint.Port != testCase.port || endpoint.Path != testCase.remotePath {
				t.Fatalf("remote endpoint = %+v, want protocol %q, host backup.example, port %d, path %q", endpoint, testCase.protocol, testCase.port, testCase.remotePath)
			}
			if request.SourceRoot != localSource {
				t.Fatalf("source root = %q, want %q", request.SourceRoot, localSource)
			}
			if endpoint.SanitizedURL() == "" || strings.Contains(endpoint.SanitizedURL(), "@") {
				t.Fatalf("sanitized endpoint URL = %q, want a nonempty URL without identity", endpoint.SanitizedURL())
			}
			if _, found := request.LocalRoots(); found {
				t.Fatal("remote request was also represented as two local roots")
			}
		})
	}
}

func TestParseWatchCommandRequestRejectsRemoteSourcesAndMalformedDestinations(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		contains  string
		secret    string
	}{
		{name: "remote source", arguments: []string{"watch", "ftp://server/source", filepath.Join(t.TempDir(), "destination")}, contains: "source"},
		{name: "unsupported destination scheme", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "https://server/archive"}, contains: "ftp or sftp"},
		{name: "single-letter unsupported scheme", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "x://server/archive"}, contains: "ftp or sftp"},
		{name: "missing server", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp:///archive"}, contains: "server host"},
		{name: "opaque URL path", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp:relative/archive"}, contains: "explicit server and directory"},
		{name: "missing path", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server"}, contains: "absolute remote directory"},
		{name: "root destination", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/"}, contains: "server root"},
		{name: "empty explicit port", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server:/archive"}, contains: "port"},
		{name: "zero port", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server:0/archive"}, contains: "port"},
		{name: "out-of-range port", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server:65536/archive"}, contains: "port"},
		{name: "dot segment", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/a/../archive"}, contains: "dot segments"},
		{name: "encoded dot segment", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/%2e%2e/archive"}, contains: "dot segments"},
		{name: "double-encoded dot segment", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/%252e%252e/archive"}, contains: "dot segments"},
		{name: "encoded slash", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/a%2fb/archive"}, contains: "separator"},
		{name: "double-encoded slash", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/a%252fb/archive"}, contains: "separator"},
		{name: "encoded backslash", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "sftp://server/a%5Cb/archive"}, contains: "separator"},
		{name: "duplicate separators", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "sftp://server/a//archive"}, contains: "empty segment"},
		{name: "trailing separator", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "sftp://server/archive/"}, contains: "trailing separator"},
		{name: "userinfo", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://secret-user:secret-password@server/archive"}, contains: "credentials", secret: "secret-password"},
		{name: "query", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "sftp://server/archive?token=secret"}, contains: "query", secret: "secret"},
		{name: "fragment", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "sftp://server/archive#secret"}, contains: "fragment", secret: "secret"},
		{name: "empty query", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/archive?"}, contains: "query"},
		{name: "empty fragment", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/archive#"}, contains: "fragment"},
		{name: "invalid percent escape", arguments: []string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/archive%2"}, contains: "syntax"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := parseWatchCommandRequest(testCase.arguments)
			if err == nil {
				t.Fatal("parseWatchCommandRequest() error = nil, want endpoint rejection")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.contains)) {
				t.Fatalf("parseWatchCommandRequest() error = %q, want it to contain %q", err, testCase.contains)
			}
			if testCase.secret != "" && strings.Contains(err.Error(), testCase.secret) {
				t.Fatalf("parseWatchCommandRequest() error disclosed protected value %q: %v", testCase.secret, err)
			}
		})
	}
}

func TestParseWatchCommandRequestTreatsWindowsDrivePathsAsLocal(t *testing.T) {
	request, err := parseWatchCommandRequest([]string{"watch", `C:\source`, `D:\destination`})
	if err != nil {
		t.Fatalf("parseWatchCommandRequest(Windows paths) error = %v", err)
	}
	if request.Destination.Kind != WatchDestinationLocal {
		t.Fatalf("destination kind = %q, want local", request.Destination.Kind)
	}
}

func TestParseWatchCommandRejectsRemoteDestinationInsteadOfCoercingItToAPath(t *testing.T) {
	_, err := parseWatchCommand([]string{"watch", filepath.Join(t.TempDir(), "source"), "ftp://server/archive"})
	if err == nil || !strings.Contains(err.Error(), "remote destination") {
		t.Fatalf("parseWatchCommand(remote) error = %v, want explicit remote/local type error", err)
	}
}
