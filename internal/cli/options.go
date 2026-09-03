package cli

import "github.com/fuse/pmx/internal/config"

type options struct {
	configPath string
}

func (o *options) loadConfig() (*config.File, error) {
	return config.Load(o.configPath)
}

func (o *options) resolveSessionPath(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	cfgFile, err := o.loadConfig()
	if err != nil {
		return "", err
	}
	if cfgFile.SessionFile != "" {
		return cfgFile.SessionFile, nil
	}
	return "", nil
}
