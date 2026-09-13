package gosync_test

import (
	"strings"
	"testing"
)

func TestLogSanitizerReplacesEveryRegisteredProtectedValue(t *testing.T) {
	testCases := []struct {
		name           string
		protectedValue string
		message        string
		safeContext    string
	}{
		{
			name:           "credential",
			protectedValue: "backup-user:password123",
			message:        "authentication failed for backup-user:password123",
			safeContext:    "authentication failed for",
		},
		{
			name:           "authentication secret",
			protectedValue: "Bearer abc123secret",
			message:        "request rejected with Bearer abc123secret",
			safeContext:    "request rejected with",
		},
		{
			name:           "cryptographic key",
			protectedValue: "private-key-material",
			message:        "key loading failed for private-key-material",
			safeContext:    "key loading failed for",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sanitizer := newLogSanitizer()
			sanitizer.RegisterProtectedValue(testCase.protectedValue)

			got := sanitizer.SanitizeMessage(testCase.message)
			if strings.Contains(got, testCase.protectedValue) {
				t.Fatalf("sanitized message = %q, contains protected value", got)
			}
			if !strings.Contains(got, "[REDACTED]") {
				t.Fatalf("sanitized message = %q, want redaction marker", got)
			}
			if !strings.Contains(got, testCase.safeContext) {
				t.Fatalf("sanitized message = %q, want safe context %q", got, testCase.safeContext)
			}
		})
	}
}

func TestLogSanitizerSanitizesEntryContextAndExcludesFileContents(t *testing.T) {
	sanitizer := newLogSanitizer()
	sanitizer.RegisterProtectedValue("top-secret")
	entry := LogEntry{
		Message: "copy failed for /sync/top-secret/notes.txt",
		Context: map[string]string{
			"path":         "/sync/top-secret/notes.txt",
			"name":         "notes-top-secret.txt",
			"error":        "permission denied: top-secret",
			"file_content": "file body must not be logged",
		},
	}

	sanitizedEntry := sanitizer.SanitizeEntry(entry)
	if strings.Contains(sanitizedEntry.Message, "top-secret") {
		t.Fatalf("sanitized message = %q, contains protected value", sanitizedEntry.Message)
	}
	if !strings.Contains(sanitizedEntry.Message, "copy failed for") {
		t.Fatalf("sanitized message = %q, lost useful operation context", sanitizedEntry.Message)
	}
	if got := sanitizedEntry.Context["path"]; !strings.Contains(got, "/sync/") || strings.Contains(got, "top-secret") {
		t.Fatalf("sanitized path = %q, want safe path context without protected value", got)
	}
	if got := sanitizedEntry.Context["error"]; !strings.Contains(got, "permission denied") || strings.Contains(got, "top-secret") {
		t.Fatalf("sanitized error = %q, want useful error context without protected value", got)
	}
	if _, containsFileContent := sanitizedEntry.Context["file_content"]; containsFileContent {
		t.Fatal("sanitized context must exclude file contents")
	}
}
