package gosync

import (
	"sort"
	"strings"
)

const LogRedactionMarker = "[REDACTED]"

type LogSanitizer struct {
	protectedValues []string
}

func newLogSanitizer() *LogSanitizer {
	return &LogSanitizer{}
}

func (sanitizer *LogSanitizer) RegisterProtectedValue(protectedValue string) {
	if sanitizer == nil || protectedValue == "" {
		return
	}

	sanitizer.protectedValues = append(sanitizer.protectedValues, protectedValue)
}

func (sanitizer *LogSanitizer) SanitizeMessage(message string) string {
	if sanitizer == nil || len(sanitizer.protectedValues) == 0 {
		return message
	}

	protectedValues := append([]string(nil), sanitizer.protectedValues...)
	sort.SliceStable(protectedValues, func(firstIndex, secondIndex int) bool {
		return len(protectedValues[firstIndex]) > len(protectedValues[secondIndex])
	})

	sanitizedMessage := message
	for _, protectedValue := range protectedValues {
		sanitizedMessage = strings.ReplaceAll(sanitizedMessage, protectedValue, LogRedactionMarker)
	}
	return sanitizedMessage
}

func (sanitizer *LogSanitizer) SanitizeEntry(entry LogEntry) LogEntry {
	sanitizedEntry := entry
	sanitizedEntry.Message = sanitizer.SanitizeMessage(entry.Message)

	if entry.Context == nil {
		return sanitizedEntry
	}

	sanitizedEntry.Context = make(map[string]string, len(entry.Context))
	for contextKey, contextValue := range entry.Context {
		if isFileContentContextKey(contextKey) {
			continue
		}
		sanitizedEntry.Context[contextKey] = sanitizer.SanitizeMessage(contextValue)
	}

	return sanitizedEntry
}

func isFileContentContextKey(contextKey string) bool {
	normalizedKey := strings.ToLower(contextKey)
	normalizedKey = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalizedKey)

	switch normalizedKey {
	case "content", "contents", "filecontent", "filecontents":
		return true
	default:
		return false
	}
}
