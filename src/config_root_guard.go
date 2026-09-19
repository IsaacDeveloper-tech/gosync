package gosync

import "fmt"

func validateConfigurationPath(configurationPath string, roots RootPaths) error {
	normalizedConfigurationPath, err := normalizeRootPath(configurationPath)
	if err != nil {
		return fmt.Errorf("normalize configuration path: %w", err)
	}
	for _, root := range []string{roots.First, roots.Second} {
		normalizedRoot, err := normalizeRootPath(root)
		if err != nil {
			return fmt.Errorf("normalize synchronization root: %w", err)
		}
		containsConfiguration, err := rootContains(normalizedRoot, normalizedConfigurationPath)
		if err != nil {
			return err
		}
		if containsConfiguration {
			return fmt.Errorf("configuration path %q is inside synchronization root %q", configurationPath, root)
		}
	}
	return nil
}
