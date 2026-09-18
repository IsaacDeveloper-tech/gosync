package gosync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var ErrConfigurationCancelled = errors.New("configuration cancelled")

func parseSynchronizationInterval(answer string) (int, error) {
	trimmedAnswer := strings.TrimSpace(answer)
	if trimmedAnswer == "" {
		return 0, errors.New("interval must be a whole number")
	}
	for _, character := range trimmedAnswer {
		if character < '0' || character > '9' {
			return 0, errors.New("interval must be a whole number")
		}
	}

	interval, err := strconv.Atoi(trimmedAnswer)
	if err != nil {
		return 0, fmt.Errorf("parse interval: %w", err)
	}
	if interval < MinimumSynchronizationIntervalSeconds || interval > MaximumSynchronizationIntervalSeconds {
		return 0, fmt.Errorf("interval must be between %d and %d seconds", MinimumSynchronizationIntervalSeconds, MaximumSynchronizationIntervalSeconds)
	}
	return interval, nil
}

func parseSynchronizationMode(answer string) (SynchronizationMode, error) {
	switch strings.TrimSpace(answer) {
	case string(SynchronizationModeBidirectional):
		return SynchronizationModeBidirectional, nil
	case string(SynchronizationModeUnidirectional):
		return SynchronizationModeUnidirectional, nil
	case string(SynchronizationModeBackup):
		return "", errors.New("BACKUP mode is not available")
	default:
		return "", fmt.Errorf("unsupported synchronization mode %q", strings.TrimSpace(answer))
	}
}

func collectConfigurationDraft(input io.Reader, output io.Writer) (ConfigurationDraft, error) {
	if input == nil || output == nil {
		return ConfigurationDraft{}, errors.New("configuration input and output are required")
	}
	return collectConfigurationDraftWithReader(bufio.NewReader(input), output)
}

func collectConfigurationDraftWithReader(reader *bufio.Reader, output io.Writer) (ConfigurationDraft, error) {
	var draft ConfigurationDraft
	for {
		if _, err := fmt.Fprint(output, "Synchronization interval in seconds (1-86400): "); err != nil {
			return ConfigurationDraft{}, fmt.Errorf("write interval prompt: %w", err)
		}
		answer, err := readConfigurationAnswer(reader)
		if err != nil {
			return ConfigurationDraft{}, configurationInputError(err)
		}
		interval, err := parseSynchronizationInterval(answer)
		if err != nil {
			if _, writeErr := fmt.Fprintf(output, "Invalid interval: %v. Please try again.\n", err); writeErr != nil {
				return ConfigurationDraft{}, fmt.Errorf("write interval validation: %w", writeErr)
			}
			continue
		}
		draft.IntervalSeconds = interval
		break
	}

	for {
		if _, err := fmt.Fprint(output, "Synchronization mode (bidirectional, unidirectional, BACKUP [unavailable]): "); err != nil {
			return ConfigurationDraft{}, fmt.Errorf("write mode prompt: %w", err)
		}
		answer, err := readConfigurationAnswer(reader)
		if err != nil {
			return ConfigurationDraft{}, configurationInputError(err)
		}
		mode, err := parseSynchronizationMode(answer)
		if err != nil {
			if _, writeErr := fmt.Fprintf(output, "Invalid mode: %v. Please try again.\n", err); writeErr != nil {
				return ConfigurationDraft{}, fmt.Errorf("write mode validation: %w", writeErr)
			}
			continue
		}
		draft.Mode = mode
		break
	}

	return draft, nil
}

func confirmConfiguration(input io.Reader, output io.Writer, draft ConfigurationDraft) (bool, error) {
	if input == nil || output == nil {
		return false, errors.New("configuration input and output are required")
	}
	return confirmConfigurationWithReader(bufio.NewReader(input), output, draft)
}

func confirmConfigurationWithReader(reader *bufio.Reader, output io.Writer, draft ConfigurationDraft) (bool, error) {
	if draft.IntervalSeconds < MinimumSynchronizationIntervalSeconds || draft.IntervalSeconds > MaximumSynchronizationIntervalSeconds {
		return false, errors.New("cannot confirm an invalid interval")
	}
	if draft.Mode != SynchronizationModeBidirectional && draft.Mode != SynchronizationModeUnidirectional {
		return false, errors.New("cannot confirm an invalid synchronization mode")
	}

	if _, err := fmt.Fprintf(output, "Configuration summary:\n  interval: %d seconds\n  mode: %s\n", draft.IntervalSeconds, draft.Mode); err != nil {
		return false, fmt.Errorf("write configuration summary: %w", err)
	}
	for {
		if _, err := fmt.Fprint(output, "Save configuration? (yes/no): "); err != nil {
			return false, fmt.Errorf("write confirmation prompt: %w", err)
		}
		answer, err := readConfigurationAnswer(reader)
		if err != nil {
			return false, configurationInputError(err)
		}
		switch strings.ToLower(answer) {
		case "yes", "y":
			return true, nil
		case "no", "n":
			return false, ErrConfigurationCancelled
		default:
			if _, writeErr := fmt.Fprintln(output, "Please answer yes or no."); writeErr != nil {
				return false, fmt.Errorf("write confirmation validation: %w", writeErr)
			}
		}
	}
}

func runInteractiveConfiguration(input io.Reader, output io.Writer) (ConfigurationSnapshot, error) {
	if input == nil || output == nil {
		return ConfigurationSnapshot{}, errors.New("configuration input and output are required")
	}
	reader := bufio.NewReader(input)
	draft, err := collectConfigurationDraftWithReader(reader, output)
	if err != nil {
		return ConfigurationSnapshot{}, err
	}
	confirmed, err := confirmConfigurationWithReader(reader, output, draft)
	if err != nil {
		return ConfigurationSnapshot{}, err
	}
	if !confirmed {
		return ConfigurationSnapshot{}, ErrConfigurationCancelled
	}

	return ConfigurationSnapshot{
		SchemaVersion:                  ConfigurationSchemaVersion,
		SynchronizationIntervalSeconds: draft.IntervalSeconds,
		SynchronizationMode:            draft.Mode,
	}, nil
}

func readConfigurationAnswer(reader *bufio.Reader) (string, error) {
	answer, err := reader.ReadString('\n')
	if len(answer) > 0 {
		return strings.TrimSpace(answer), nil
	}
	return "", err
}

func configurationInputError(err error) error {
	if errors.Is(err, io.EOF) {
		return ErrConfigurationCancelled
	}
	return fmt.Errorf("read configuration input: %w", err)
}
