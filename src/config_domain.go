package gosync

import "fmt"

const (
	LegacyConfigurationSchemaVersion      = 1
	ConfigurationSchemaVersion            = 2
	BackupConfigurationSchemaVersion      = 2
	MinimumSynchronizationIntervalSeconds = 1
	MaximumSynchronizationIntervalSeconds = 86400
	MinimumBackupRetentionCount           = 1
	MaximumBackupRetentionCount           = 100
)

type SynchronizationMode string

const (
	SynchronizationModeBidirectional  SynchronizationMode = "bidirectional"
	SynchronizationModeUnidirectional SynchronizationMode = "unidirectional"
	SynchronizationModeBackup         SynchronizationMode = "backup"
)

type ConfigurationSnapshot struct {
	SchemaVersion                  int
	SynchronizationIntervalSeconds int
	SynchronizationMode            SynchronizationMode
	BackupRetentionCount           int
}

type ConfigurationDraft struct {
	IntervalSeconds      int
	Mode                 SynchronizationMode
	BackupRetentionCount int
}

func validateConfigurationSnapshot(configuration ConfigurationSnapshot) error {
	if configuration.SchemaVersion != LegacyConfigurationSchemaVersion && configuration.SchemaVersion != ConfigurationSchemaVersion {
		return fmt.Errorf("unsupported configuration schema version %d", configuration.SchemaVersion)
	}
	if configuration.SynchronizationIntervalSeconds < MinimumSynchronizationIntervalSeconds || configuration.SynchronizationIntervalSeconds > MaximumSynchronizationIntervalSeconds {
		return fmt.Errorf("synchronization interval must be between %d and %d seconds", MinimumSynchronizationIntervalSeconds, MaximumSynchronizationIntervalSeconds)
	}
	if configuration.SchemaVersion == LegacyConfigurationSchemaVersion {
		if configuration.BackupRetentionCount != 0 {
			return fmt.Errorf("legacy configuration cannot define backup retention")
		}
		if configuration.SynchronizationMode != SynchronizationModeBidirectional && configuration.SynchronizationMode != SynchronizationModeUnidirectional {
			return fmt.Errorf("unsupported synchronization mode %q", configuration.SynchronizationMode)
		}
		return nil
	}

	switch configuration.SynchronizationMode {
	case SynchronizationModeBidirectional, SynchronizationModeUnidirectional:
		if configuration.BackupRetentionCount != 0 {
			return fmt.Errorf("backup retention is only valid for BACKUP mode")
		}
	case SynchronizationModeBackup:
		if configuration.BackupRetentionCount < MinimumBackupRetentionCount || configuration.BackupRetentionCount > MaximumBackupRetentionCount {
			return fmt.Errorf("backup retention must be between %d and %d versions", MinimumBackupRetentionCount, MaximumBackupRetentionCount)
		}
	default:
		return fmt.Errorf("unsupported synchronization mode %q", configuration.SynchronizationMode)
	}
	return nil
}
