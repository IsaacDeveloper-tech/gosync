package gosync

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type RemoteProtocol string

const (
	RemoteProtocolFTP  RemoteProtocol = "ftp"
	RemoteProtocolSFTP RemoteProtocol = "sftp"
)

type WatchDestinationKind string

const (
	WatchDestinationLocal  WatchDestinationKind = "local"
	WatchDestinationRemote WatchDestinationKind = "remote"
)

// RemoteEndpoint contains only validated, non-secret connection coordinates.
// Credentials and the original URL are deliberately not retained.
type RemoteEndpoint struct {
	Protocol RemoteProtocol
	Host     string
	Port     int
	Path     string
}

type WatchDestination struct {
	Kind           WatchDestinationKind
	LocalPath      string
	RemoteEndpoint RemoteEndpoint
}

type WatchCommandRequest struct {
	SourceRoot  string
	Destination WatchDestination
}

func (request WatchCommandRequest) LocalRoots() (RootPaths, bool) {
	if request.Destination.Kind != WatchDestinationLocal {
		return RootPaths{}, false
	}
	return RootPaths{First: request.SourceRoot, Second: request.Destination.LocalPath}, true
}

func (endpoint RemoteEndpoint) SanitizedURL() string {
	if endpoint.Protocol == "" || endpoint.Host == "" || endpoint.Port < 1 || endpoint.Port > 65535 || endpoint.Path == "" {
		return ""
	}
	return (&url.URL{
		Scheme: string(endpoint.Protocol),
		Host:   net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port)),
		Path:   endpoint.Path,
	}).String()
}

func (endpoint RemoteEndpoint) String() string {
	return endpoint.SanitizedURL()
}

func looksLikeURLArgument(value string) bool {
	if isWindowsDrivePath(value) {
		return false
	}

	colonIndex := strings.IndexByte(value, ':')
	if colonIndex <= 0 || !isValidURIScheme(value[:colonIndex]) {
		return false
	}
	scheme := strings.ToLower(value[:colonIndex])
	remainder := value[colonIndex+1:]
	return scheme == string(RemoteProtocolFTP) || scheme == string(RemoteProtocolSFTP) ||
		strings.HasPrefix(remainder, "//") || strings.HasPrefix(remainder, "/") ||
		strings.HasPrefix(remainder, "?") || strings.HasPrefix(remainder, "#")
}

func isWindowsDrivePath(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	if !((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) {
		return false
	}
	return !strings.HasPrefix(value[2:], "//")
}

func isValidURIScheme(scheme string) bool {
	if scheme == "" || !isASCIIAlpha(scheme[0]) {
		return false
	}
	for index := 1; index < len(scheme); index++ {
		character := scheme[index]
		if !isASCIIAlpha(character) && (character < '0' || character > '9') && character != '+' && character != '.' && character != '-' {
			return false
		}
	}
	return true
}

func isASCIIAlpha(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
}

func parseRemoteEndpoint(endpointText string) (RemoteEndpoint, error) {
	if strings.ContainsAny(endpointText, " \t\r\n") {
		return RemoteEndpoint{}, fmt.Errorf("URL contains unescaped whitespace")
	}

	parsedURL, err := url.Parse(endpointText)
	if err != nil {
		return RemoteEndpoint{}, fmt.Errorf("URL syntax is invalid")
	}
	protocol := RemoteProtocol(strings.ToLower(parsedURL.Scheme))
	if protocol != RemoteProtocolFTP && protocol != RemoteProtocolSFTP {
		return RemoteEndpoint{}, fmt.Errorf("scheme must be ftp or sftp")
	}
	if parsedURL.Opaque != "" {
		return RemoteEndpoint{}, fmt.Errorf("URL must use an explicit server and directory path")
	}
	if parsedURL.Host == "" || parsedURL.Hostname() == "" {
		return RemoteEndpoint{}, fmt.Errorf("server host is required")
	}
	if parsedURL.User != nil {
		return RemoteEndpoint{}, fmt.Errorf("credentials must not be included in the URL")
	}
	if parsedURL.RawQuery != "" || parsedURL.ForceQuery || strings.Contains(endpointText, "?") {
		return RemoteEndpoint{}, fmt.Errorf("query strings are not allowed")
	}
	if strings.Contains(endpointText, "#") {
		return RemoteEndpoint{}, fmt.Errorf("fragments are not allowed")
	}

	port, err := parseRemotePort(parsedURL)
	if err != nil {
		return RemoteEndpoint{}, err
	}
	remotePath, err := validateRemoteDirectoryPath(parsedURL.Path)
	if err != nil {
		return RemoteEndpoint{}, err
	}
	if err := validateRemotePathEscaping(parsedURL.EscapedPath()); err != nil {
		return RemoteEndpoint{}, err
	}

	return RemoteEndpoint{
		Protocol: protocol,
		Host:     strings.ToLower(parsedURL.Hostname()),
		Port:     port,
		Path:     remotePath,
	}, nil
}

func parseRemotePort(parsedURL *url.URL) (int, error) {
	portText := parsedURL.Port()
	if portText == "" {
		if strings.HasSuffix(parsedURL.Host, ":") {
			return 0, fmt.Errorf("server port must not be empty")
		}
		if RemoteProtocol(strings.ToLower(parsedURL.Scheme)) == RemoteProtocolFTP {
			return 21, nil
		}
		return 22, nil
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("server port must be between 1 and 65535")
	}
	return port, nil
}

func validateRemoteDirectoryPath(remotePath string) (string, error) {
	if remotePath == "" || !strings.HasPrefix(remotePath, "/") {
		return "", fmt.Errorf("an absolute remote directory path is required")
	}
	if remotePath == "/" || strings.HasPrefix(remotePath, "//") {
		return "", fmt.Errorf("the server root is not a valid destination")
	}
	if strings.Contains(remotePath, "\\") || !utf8.ValidString(remotePath) {
		return "", fmt.Errorf("remote directory path contains an unsupported separator or encoding")
	}
	for _, character := range remotePath {
		if unicode.IsControl(character) {
			return "", fmt.Errorf("remote directory path contains a control character")
		}
	}
	if strings.HasSuffix(remotePath, "/") {
		return "", fmt.Errorf("remote directory path must not have a trailing separator")
	}

	for _, segment := range strings.Split(strings.TrimPrefix(remotePath, "/"), "/") {
		if segment == "" {
			return "", fmt.Errorf("remote directory path contains an empty segment")
		}
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("remote directory path must not contain dot segments")
		}
	}
	return remotePath, nil
}

func validateRemotePathEscaping(escapedPath string) error {
	candidate := escapedPath
	for {
		lowerCandidate := strings.ToLower(candidate)
		if strings.Contains(lowerCandidate, "%2f") || strings.Contains(lowerCandidate, "%5c") {
			return fmt.Errorf("encoded path separators are not allowed")
		}
		decodedCandidate, err := url.PathUnescape(candidate)
		if err != nil {
			return nil
		}
		if _, err := validateRemoteDirectoryPath(decodedCandidate); err != nil {
			return fmt.Errorf("remote directory path has ambiguous escaping: %w", err)
		}
		if decodedCandidate == candidate {
			return nil
		}
		candidate = decodedCandidate
	}
}
