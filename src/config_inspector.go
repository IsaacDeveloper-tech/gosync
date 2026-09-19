package gosync

import (
	"errors"
	"fmt"
	"os"
)

var ErrUnsupportedConfigurationEntry = errors.New("unsupported configuration filesystem entry")

type ConfigurationFileStatus string

const (
	ConfigurationFileMissing     ConfigurationFileStatus = "missing"
	ConfigurationFileRegular     ConfigurationFileStatus = "regular"
	ConfigurationFileUnsupported ConfigurationFileStatus = "unsupported"
)

func inspectConfigurationFile(path string) (ConfigurationFileStatus, error) {
	fileInformation, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConfigurationFileMissing, nil
		}
		return "", fmt.Errorf("inspect configuration file: %w", err)
	}
	if !fileInformation.Mode().IsRegular() {
		return ConfigurationFileUnsupported, fmt.Errorf("%w: %s", ErrUnsupportedConfigurationEntry, path)
	}
	return ConfigurationFileRegular, nil
}

type ConfigurationLoadStatus string

const (
	ConfigurationLoadMissing     ConfigurationLoadStatus = "missing"
	ConfigurationLoadValid       ConfigurationLoadStatus = "valid"
	ConfigurationLoadInvalid     ConfigurationLoadStatus = "invalid"
	ConfigurationLoadUnsupported ConfigurationLoadStatus = "unsupported"
	ConfigurationLoadIOFailure   ConfigurationLoadStatus = "io_failure"
)

type ConfigurationLoadResult struct {
	Status        ConfigurationLoadStatus
	Configuration ConfigurationSnapshot
}

func loadConfiguration(path string) (ConfigurationLoadResult, error) {
	fileStatus, err := inspectConfigurationFile(path)
	if err != nil {
		if fileStatus == ConfigurationFileUnsupported {
			return ConfigurationLoadResult{Status: ConfigurationLoadUnsupported}, err
		}
		return ConfigurationLoadResult{Status: ConfigurationLoadIOFailure}, err
	}
	if fileStatus == ConfigurationFileMissing {
		return ConfigurationLoadResult{Status: ConfigurationLoadMissing}, nil
	}

	encodedConfiguration, err := os.ReadFile(path)
	if err != nil {
		return ConfigurationLoadResult{Status: ConfigurationLoadIOFailure}, fmt.Errorf("read configuration file: %w", err)
	}
	configuration, err := decodeConfiguration(string(encodedConfiguration))
	if err != nil {
		return ConfigurationLoadResult{Status: ConfigurationLoadInvalid}, fmt.Errorf("decode configuration file: %w", err)
	}
	return ConfigurationLoadResult{Status: ConfigurationLoadValid, Configuration: configuration}, nil
}
