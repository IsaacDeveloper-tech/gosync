package gosync

import (
	"errors"
	"fmt"
	"io"
)

type ConfigurationServiceOptions struct {
	Store  ConfigurationStore
	Input  io.Reader
	Output io.Writer
	Logger *LoggingCoordinator
}

type ConfigurationService struct {
	store  ConfigurationStore
	input  io.Reader
	output io.Writer
	logger *LoggingCoordinator
}

func newConfigurationService(options ConfigurationServiceOptions) *ConfigurationService {
	return &ConfigurationService{
		store:  options.Store,
		input:  options.Input,
		output: options.Output,
		logger: options.Logger,
	}
}

func (service *ConfigurationService) Configure() (ConfigurationSnapshot, error) {
	if service == nil {
		return ConfigurationSnapshot{}, errors.New("configuration service is required")
	}
	if err := service.logLifecycleEvent(LogEventConfigurationStarted, LogSeverityInfo, "configuration started", nil); err != nil {
		return ConfigurationSnapshot{}, err
	}

	ownership, err := acquireConfigurationOwnership(service.store.Path())
	if err != nil {
		return service.failConfiguration(err)
	}
	defer ownership.Release()

	loadResult, loadErr := service.store.Load()
	if loadErr != nil {
		if loadResult.Status != ConfigurationLoadInvalid {
			return service.failConfiguration(loadErr)
		}
		if _, writeErr := fmt.Fprintf(service.output, "Existing configuration is invalid: %v\n", loadErr); writeErr != nil {
			return service.failConfiguration(writeErr)
		}
	}

	configuration, err := runInteractiveConfiguration(service.input, service.output)
	if err != nil {
		if errors.Is(err, ErrConfigurationCancelled) {
			if logErr := service.logLifecycleEvent(LogEventConfigurationCancelled, LogSeverityWarn, "configuration cancelled", nil); logErr != nil {
				return ConfigurationSnapshot{}, logErr
			}
			return ConfigurationSnapshot{}, err
		}
		return service.failConfiguration(err)
	}
	if err := service.store.Save(configuration); err != nil {
		return service.failConfiguration(err)
	}
	if err := service.logLifecycleEvent(LogEventConfigurationSucceeded, LogSeverityInfo, "configuration saved successfully", nil); err != nil {
		return ConfigurationSnapshot{}, err
	}
	return configuration, nil
}

func (service *ConfigurationService) failConfiguration(configurationError error) (ConfigurationSnapshot, error) {
	if logErr := service.logLifecycleEvent(
		LogEventConfigurationFailed,
		LogSeverityError,
		"configuration failed",
		map[string]string{"error": configurationError.Error()},
	); logErr != nil {
		return ConfigurationSnapshot{}, logErr
	}
	return ConfigurationSnapshot{}, configurationError
}

func (service *ConfigurationService) logLifecycleEvent(event LogEvent, severity LogSeverity, message string, context map[string]string) error {
	if service.logger == nil {
		return nil
	}
	return service.logger.Log(LogEntry{Severity: severity, Event: event, Message: message, Context: context})
}
