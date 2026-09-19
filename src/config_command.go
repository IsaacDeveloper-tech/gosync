package gosync

import (
	"io"
	"os"
)

type ConfigureCommandOptions struct {
	Input  io.Reader
	Output io.Writer
	Store  *ConfigurationStore
	Logger *LoggingCoordinator
}

func runConfigureCommand(arguments []string, options ConfigureCommandOptions) error {
	if err := parseConfigureCommand(arguments); err != nil {
		return err
	}
	input := options.Input
	if input == nil {
		input = os.Stdin
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}
	store := options.Store
	if store == nil {
		configurationPath, err := resolveConfigurationFilePath()
		if err != nil {
			return err
		}
		defaultStore := newConfigurationStore(configurationPath)
		store = &defaultStore
	}
	logger := options.Logger
	if logger == nil {
		var err error
		logger, err = newLoggingCoordinator(LoggingCoordinatorOptions{ConsoleWriter: output})
		if err != nil {
			return err
		}
	}
	service := newConfigurationService(ConfigurationServiceOptions{
		Store:  *store,
		Input:  input,
		Output: output,
		Logger: logger,
	})
	_, err := service.Configure()
	return err
}
