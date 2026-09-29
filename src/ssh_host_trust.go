package gosync

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SSHHostTrustStatus string

const (
	SSHHostTrustRecognized  SSHHostTrustStatus = "recognized"
	SSHHostTrustAbsent      SSHHostTrustStatus = "absent"
	SSHHostTrustChanged     SSHHostTrustStatus = "changed"
	SSHHostTrustUnavailable SSHHostTrustStatus = "unavailable"
)

type RemoteSSHHostKey struct {
	Algorithm string
	Data      []byte
}

type SSHHostTrustVerifier interface {
	CheckHostKey(host string, port int, serverKey RemoteSSHHostKey) (SSHHostTrustStatus, error)
}

type SSHHostTrustStore struct {
	Path string
}

func newSSHHostTrustStore() (SSHHostTrustStore, error) {
	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return SSHHostTrustStore{}, fmt.Errorf("resolve user SSH trust location: %w", err)
	}
	return SSHHostTrustStore{Path: filepath.Join(userHomeDirectory, ".ssh", "known_hosts")}, nil
}

func newSSHHostTrustStoreAt(path string) SSHHostTrustStore {
	return SSHHostTrustStore{Path: path}
}

func (store SSHHostTrustStore) CheckHostKey(host string, port int, serverKey RemoteSSHHostKey) (SSHHostTrustStatus, error) {
	if store.Path == "" {
		return SSHHostTrustUnavailable, errors.New("SSH host trust location is unavailable")
	}
	if strings.TrimSpace(host) == "" || port < 1 || port > 65535 || serverKey.Algorithm == "" || len(serverKey.Data) == 0 {
		return SSHHostTrustUnavailable, errors.New("SSH host and port identity or server key is invalid")
	}

	knownHostsFile, err := os.Open(store.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return SSHHostTrustAbsent, nil
		}
		return SSHHostTrustUnavailable, fmt.Errorf("open SSH host trust file: %w", err)
	}

	candidates := sshKnownHostsCandidates(host, port)
	matchedHost := false
	trustedKey := false
	revokedKey := false
	scanner := bufio.NewScanner(knownHostsFile)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		marker, hostPatterns, keyAlgorithm, encodedKey, found := parseOpenSSHKnownHostLine(scanner.Text())
		if !found || !sshKnownHostPatternsMatch(hostPatterns, candidates) {
			continue
		}
		matchedHost = true
		keyData, decodeErr := base64.StdEncoding.DecodeString(encodedKey)
		if decodeErr != nil {
			keyData, decodeErr = base64.RawStdEncoding.DecodeString(encodedKey)
		}
		if decodeErr != nil || keyAlgorithm != serverKey.Algorithm || !bytes.Equal(keyData, serverKey.Data) {
			continue
		}
		if marker == "@revoked" {
			revokedKey = true
			continue
		}
		if marker == "" {
			trustedKey = true
		}
	}
	scanErr := scanner.Err()
	closeErr := knownHostsFile.Close()
	if scanErr != nil || closeErr != nil {
		return SSHHostTrustUnavailable, fmt.Errorf("read SSH host trust file: %w", errors.Join(scanErr, closeErr))
	}
	if revokedKey {
		return SSHHostTrustChanged, nil
	}
	if trustedKey {
		return SSHHostTrustRecognized, nil
	}
	if matchedHost {
		return SSHHostTrustChanged, nil
	}
	return SSHHostTrustAbsent, nil
}

func sshKnownHostsCandidates(host string, port int) []string {
	normalizedHost := strings.ToLower(host)
	portSpecificHost := fmt.Sprintf("[%s]:%d", normalizedHost, port)
	if port != 22 {
		return []string{portSpecificHost, fmt.Sprintf("[%s]:%d", strings.ToUpper(host), port)}
	}
	return []string{normalizedHost, strings.ToUpper(host), portSpecificHost, fmt.Sprintf("[%s]:22", strings.ToUpper(host))}
}

func parseOpenSSHKnownHostLine(line string) (marker, hostPatterns, keyAlgorithm, encodedKey string, found bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
		return "", "", "", "", false
	}
	if strings.HasPrefix(fields[0], "@") {
		marker = fields[0]
		fields = fields[1:]
	}
	if len(fields) < 3 {
		return "", "", "", "", false
	}
	return marker, fields[0], fields[1], fields[2], true
}

func sshKnownHostPatternsMatch(hostPatterns string, candidates []string) bool {
	matchedPositive := false
	for _, hostPattern := range strings.Split(hostPatterns, ",") {
		if hostPattern == "" {
			continue
		}
		negated := strings.HasPrefix(hostPattern, "!")
		pattern := strings.TrimPrefix(hostPattern, "!")
		matched := false
		for _, candidate := range candidates {
			if sshKnownHostPatternMatches(pattern, candidate) {
				matched = true
				break
			}
		}
		if matched && negated {
			return false
		}
		if matched {
			matchedPositive = true
		}
	}
	return matchedPositive
}

func sshKnownHostPatternMatches(pattern, candidate string) bool {
	if strings.HasPrefix(pattern, "|1|") {
		return hashedSSHHostMatches(pattern, candidate)
	}
	regularExpression := regexp.QuoteMeta(strings.ToLower(pattern))
	regularExpression = strings.ReplaceAll(regularExpression, `\*`, `.*`)
	regularExpression = strings.ReplaceAll(regularExpression, `\?`, `.`)
	matched, err := regexp.MatchString("^"+regularExpression+"$", strings.ToLower(candidate))
	return err == nil && matched
}

func hashedSSHHostMatches(pattern, candidate string) bool {
	parts := strings.Split(pattern, "|")
	if len(parts) != 4 || parts[1] != "1" {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expectedHash, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	mac := hmac.New(sha1.New, salt)
	_, _ = mac.Write([]byte(candidate))
	return hmac.Equal(mac.Sum(nil), expectedHash)
}
