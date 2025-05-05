package lib

import (
	"blanktiger/hm/configuration"
)

func Symlink(c *configuration.Configuration, configs []Config) error {
	for _, cfg := range configs {
		Logger.Info("symlinking", "from", cfg.From, "to", cfg.To)
		err := symlink(cfg.From, cfg.To)
		if err != nil {
			return err
		}
	}

	return nil
}

func Copy(c *configuration.Configuration, configs []Config) error {
	for _, cfg := range configs {
		Logger.Info("copying", "from", cfg.From, "to", cfg.To)
		err := copyCfg(cfg.From, cfg.To)
		if err != nil {
			return err
		}
	}
	return nil
}

func Remove(c *configuration.Configuration, configs []Config) error {
	for _, cfg := range configs {
		Logger.Info("removing config from target", "target", cfg.To)
		err := removeCfg(cfg.To)
		if err != nil {
			return err
		}
	}
	return nil
}

func Install(lock *Lockfile) map[string]installInfo {
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

		if cfg.Requirements.Install == nil {
			Logger.Debug("there is no INSTALL file for this config (probably)", "cfgName", cfg.Name)
			continue
		}
		Logger.Info("trying to install", "cfgName", cfg.Name)
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
		info.UninstallInstructions = []string{}
		forUpdate[cfg.Name] = info
	}
	return forUpdate
}

func Uninstall(lock *Lockfile) map[string]installInfo {
	forUpdate := make(map[string]installInfo)

	for _, cfg := range lock.HiddenConfigs {
		info := uninstallForCfg(cfg)
		if info != nil {
			forUpdate[cfg.Name] = *info
		}
	}

	return forUpdate
}

func InstallGlobalDependencies(dependencies *[]globalDependency) error {
	Logger.Info("installing global dependencies")

	for idx, dep := range *dependencies {
		if dep.InstallInfo.IsInstalled {
			Logger.Debug("skipping installation of an already installed global dependency", "pkgName", dep.Instruction.Pkg)
			continue
		}

		info, err := installGlobalDependency(dep)
		if err != nil {
			return err
		}
		(*dependencies)[idx].InstallInfo = info
	}

	return nil
}
