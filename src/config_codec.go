package gosync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func encodeConfiguration(configuration ConfigurationSnapshot) (string, error) {
	if err := validateConfigurationSnapshot(configuration); err != nil {
		return "", err
	}

	var encodedConfiguration []byte
	var err error
	if configuration.SchemaVersion == LegacyConfigurationSchemaVersion || configuration.SynchronizationMode != SynchronizationModeBackup {
		encodedConfiguration, err = json.Marshal(struct {
			SchemaVersion                  int                 `json:"schemaVersion"`
			SynchronizationIntervalSeconds int                 `json:"synchronizationIntervalSeconds"`
			SynchronizationMode            SynchronizationMode `json:"synchronizationMode"`
		}{
			SchemaVersion:                  configuration.SchemaVersion,
			SynchronizationIntervalSeconds: configuration.SynchronizationIntervalSeconds,
			SynchronizationMode:            configuration.SynchronizationMode,
		})
	} else {
		encodedConfiguration, err = json.Marshal(struct {
			SchemaVersion                  int                 `json:"schemaVersion"`
			SynchronizationIntervalSeconds int                 `json:"synchronizationIntervalSeconds"`
			SynchronizationMode            SynchronizationMode `json:"synchronizationMode"`
			BackupRetentionCount           int                 `json:"backupRetentionCount"`
		}{
			SchemaVersion:                  configuration.SchemaVersion,
			SynchronizationIntervalSeconds: configuration.SynchronizationIntervalSeconds,
			SynchronizationMode:            configuration.SynchronizationMode,
			BackupRetentionCount:           configuration.BackupRetentionCount,
		})
	}
	if err != nil {
		return "", fmt.Errorf("encode configuration: %w", err)
	}
	return string(encodedConfiguration), nil
}

func decodeConfiguration(encodedConfiguration string) (ConfigurationSnapshot, error) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(encodedConfiguration)))
	firstToken, err := decoder.Token()
	if err != nil {
		return ConfigurationSnapshot{}, fmt.Errorf("decode configuration object: %w", err)
	}
	objectDelimiter, ok := firstToken.(json.Delim)
	if !ok || objectDelimiter != '{' {
		return ConfigurationSnapshot{}, errorsInvalidConfigurationObject()
	}

	values := make(map[string]json.RawMessage, 4)
	for decoder.More() {
		propertyToken, err := decoder.Token()
		if err != nil {
			return ConfigurationSnapshot{}, fmt.Errorf("decode configuration property: %w", err)
		}
		propertyName, ok := propertyToken.(string)
		if !ok {
			return ConfigurationSnapshot{}, errorsInvalidConfigurationProperty()
		}
		if _, duplicate := values[propertyName]; duplicate {
			return ConfigurationSnapshot{}, fmt.Errorf("duplicate configuration property %q", propertyName)
		}
		if propertyName != "schemaVersion" && propertyName != "synchronizationIntervalSeconds" && propertyName != "synchronizationMode" && propertyName != "backupRetentionCount" {
			return ConfigurationSnapshot{}, fmt.Errorf("unknown configuration property %q", propertyName)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return ConfigurationSnapshot{}, fmt.Errorf("decode configuration property %q: %w", propertyName, err)
		}
		values[propertyName] = value
	}
	closingToken, err := decoder.Token()
	if err != nil {
		return ConfigurationSnapshot{}, fmt.Errorf("close configuration object: %w", err)
	}
	closingDelimiter, ok := closingToken.(json.Delim)
	if !ok || closingDelimiter != '}' {
		return ConfigurationSnapshot{}, errorsInvalidConfigurationObject()
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ConfigurationSnapshot{}, errorsNewTrailingConfigurationData()
		}
		return ConfigurationSnapshot{}, fmt.Errorf("decode trailing configuration data: %w", err)
	}

	for _, propertyName := range []string{"schemaVersion", "synchronizationIntervalSeconds", "synchronizationMode"} {
		if _, found := values[propertyName]; !found {
			return ConfigurationSnapshot{}, fmt.Errorf("missing configuration property %q", propertyName)
		}
	}

	schemaVersion, err := decodeConfigurationInteger(values["schemaVersion"], "schemaVersion")
	if err != nil {
		return ConfigurationSnapshot{}, err
	}
	interval, err := decodeConfigurationInteger(values["synchronizationIntervalSeconds"], "synchronizationIntervalSeconds")
	if err != nil {
		return ConfigurationSnapshot{}, err
	}
	var mode string
	if rawMode := values["synchronizationMode"]; len(rawMode) == 0 || rawMode[0] != '"' || json.Unmarshal(rawMode, &mode) != nil {
		return ConfigurationSnapshot{}, errorsInvalidConfigurationMode()
	}

	configuration := ConfigurationSnapshot{
		SchemaVersion:                  schemaVersion,
		SynchronizationIntervalSeconds: interval,
		SynchronizationMode:            SynchronizationMode(mode),
	}
	_, hasRetention := values["backupRetentionCount"]
	if schemaVersion == LegacyConfigurationSchemaVersion {
		if hasRetention {
			return ConfigurationSnapshot{}, fmt.Errorf("legacy configuration cannot define backup retention")
		}
	} else if schemaVersion == ConfigurationSchemaVersion {
		if configuration.SynchronizationMode == SynchronizationModeBackup {
			if !hasRetention {
				return ConfigurationSnapshot{}, fmt.Errorf("missing configuration property %q", "backupRetentionCount")
			}
			retentionCount, err := decodeConfigurationInteger(values["backupRetentionCount"], "backupRetentionCount")
			if err != nil {
				return ConfigurationSnapshot{}, err
			}
			configuration.BackupRetentionCount = retentionCount
		} else if hasRetention {
			return ConfigurationSnapshot{}, fmt.Errorf("backup retention is only valid for BACKUP mode")
		}
	}
	if err := validateConfigurationSnapshot(configuration); err != nil {
		return ConfigurationSnapshot{}, err
	}
	return configuration, nil
}

func decodeConfigurationInteger(rawValue json.RawMessage, propertyName string) (int, error) {
	value, err := strconv.ParseInt(string(rawValue), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("configuration property %q must be an integer: %w", propertyName, err)
	}
	convertedValue := int(value)
	if int64(convertedValue) != value {
		return 0, fmt.Errorf("configuration property %q is outside the supported integer range", propertyName)
	}
	return convertedValue, nil
}

func errorsInvalidConfigurationObject() error {
	return fmt.Errorf("configuration must be a JSON object")
}

func errorsInvalidConfigurationProperty() error {
	return fmt.Errorf("configuration property name is invalid")
}

func errorsInvalidConfigurationMode() error {
	return fmt.Errorf("configuration property %q must be a string", "synchronizationMode")
}

func errorsNewTrailingConfigurationData() error {
	return fmt.Errorf("configuration contains trailing data")
}
