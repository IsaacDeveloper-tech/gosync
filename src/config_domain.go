package gosync

import "fmt"

const (
	ConfigurationSchemaVersion            = 1
	MinimumSynchronizationIntervalSeconds = 1
	MaximumSynchronizationIntervalSeconds = 86400
)

type SynchronizationMode string

const (
	SynchronizationModeBidirectional  SynchronizationMode = "bidirectional"
	SynchronizationModeUnidirectional SynchronizationMode = "unidirectional"
	SynchronizationModeBackup         SynchronizationMode = "BACKUP"
)

type ConfigurationSnapshot struct {
	SchemaVersion                  int
	SynchronizationIntervalSeconds int
	SynchronizationMode            SynchronizationMode
}

type ConfigurationDraft struct {
	IntervalSeconds int
	Mode            SynchronizationMode
}

func validateConfigurationSnapshot(configuration ConfigurationSnapshot) error {
	if configuration.SchemaVersion != ConfigurationSchemaVersion {
		return fmt.Errorf("unsupported configuration schema version %d", configuration.SchemaVersion)
	}
	if configuration.SynchronizationIntervalSeconds < MinimumSynchronizationIntervalSeconds || configuration.SynchronizationIntervalSeconds > MaximumSynchronizationIntervalSeconds {
		return fmt.Errorf("synchronization interval must be between %d and %d seconds", MinimumSynchronizationIntervalSeconds, MaximumSynchronizationIntervalSeconds)
	}
	if configuration.SynchronizationMode != SynchronizationModeBidirectional && configuration.SynchronizationMode != SynchronizationModeUnidirectional {
		return fmt.Errorf("unsupported synchronization mode %q", configuration.SynchronizationMode)
	}
	return nil
}
