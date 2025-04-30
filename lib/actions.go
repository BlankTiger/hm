package lib

import (
	"blanktiger/hm/configuration"
)

func FindConfigsToSymlink(c *configuration.Configuration, l *lockfile) ([]config, error) {
	return l.Configs, nil
}

func Symlink(c *configuration.Configuration, configs []config) error {
	for _, cfg := range configs {
		Logger.Info("symlinking", "from", cfg.From, "to", cfg.To)
		err := symlink(cfg.From, cfg.To)
		if err != nil {
			return err
		}
	}

	return nil
}

func Copy(c *configuration.Configuration, configs []config) error {
	for _, cfg := range configs {
		Logger.Info("copying", "from", cfg.From, "to", cfg.To)
		err := copyCfg(cfg.From, cfg.To)
		if err != nil {
			return err
		}
	}
	return nil
}

func FindConfigsToRemove(c *configuration.Configuration, l *lockfile) ([]config, error) {
	return l.HiddenConfigs, nil
}

func Remove(c *configuration.Configuration, configs []config) error {
	for _, cfg := range configs {
		Logger.Info("removing config from target", "target", cfg.To)
		err := removeCfg(cfg.To)
		if err != nil {
			return err
		}
	}
	return nil
}

func Install(lock *lockfile) map[string]installInfo {
	forUpdate := make(map[string]installInfo)
	for _, cfg := range lock.Configs {
		if cfg.InstallInfo.IsInstalled {
			Logger.Debug("skipping installation of already installed packages for config", "cfgName", cfg.Name)
			continue
		}

		info := installInfo{}
		err := installDependencies(cfg.Requirements.Dependencies)
		if err != nil {
			Logger.Debug("something went wrong while installing dependencies, trying to continue", "cfgName", cfg.Name, "err", err)
			continue
		}
		info.DependenciesInstalled = true

		cmd, err := install(*cfg.Requirements.Install)
		if err != nil {
			Logger.Debug("something went wrong while installing dependencies, trying to continue", "cfgName", cfg.Name, "err", err)
			continue
		}
		info.InstallTime = now()
		info.InstallInstruction = cmd
		info.IsInstalled = true
		info.WasUninstalled = false
		info.UninstallTime = ""
		info.UninstallInstruction = ""
		forUpdate[cfg.Name] = info
	}
	return forUpdate
}
