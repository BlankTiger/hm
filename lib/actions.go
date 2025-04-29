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

func FindPkgsToInstall(c *configuration.Configuration, ld *lockfileDiff) ([]installInstruction, error) {
	instructions := []installInstruction{}
	return instructions, nil
}

func Install(c *configuration.Configuration, l *lockfile, instructions []installInstruction) error {
	for _, inst := range instructions {
		cmd, err := install(inst)
		if err != nil {
			return err
		}
	}
	return nil
}

// func Install(cfg config) (res *installInfo, err error) {
// 	res, err = &cfg.InstallInfo, nil
//
// 	if cfg.Requirements.Install == nil {
// 		Logger.Debug("not installing the pkg, because there was no INSTALL instructions", "pkg", cfg.Name)
// 		return res, nil
// 	}
//
// 	// first install the dependencies if any
// 	{
// 		err = installDependencies(cfg.Requirements.Dependencies)
// 		if err != nil {
// 			return res, err
// 		}
// 		res.DependenciesInstalled = true
// 	}
//
// 	{
// 		cmd, err := install(*cfg.Requirements.Install)
// 		if err != nil {
// 			return res, err
// 		}
// 		res.InstallInstruction = cmd
// 	}
//
// 	// info handling
// 	{
// 		res.InstallTime = now()
// 		res.IsInstalled = true
// 		res.WasUninstalled = false
// 		res.UninstallTime = ""
// 	}
// 	return res, err
// }
