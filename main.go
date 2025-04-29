package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/instructions"
	"blanktiger/hm/lib"
	"os"
	"slices"
)

func main() {
	c := conf.Parse()
	c.Display()
	c.AssertCorrectness()

	lib.Logger = c.Logger
	instructions.Init(c.Logger)
	err := _main(&c)
	if err != nil {
		c.Logger.Error("program exited with an error", "error", err)
		os.Exit(1)
	}
}

func _main(c *conf.Configuration) error {
	lockfile, err := lib.BaseLockfileCreation(c)
	if err != nil {
		c.Logger.Error("encountered an error during base lockfile creation", "error", err)
		return err
	}
	lockfileBefore := *lockfile
	defer func() {
		err := lockfile.Save(c.LockfilePath, c.DefaultIndent)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save the lockfile", "err", err)
			return
		}
	}()

	// TODO: think if this is correct, for now just reset
	*lockfile = lib.DefaultLockfile

	if c.CopyMode {
		lib.Logger.Debug("setting mode to cpy")
		lockfile.Mode = lib.Cpy
	} else {
		lib.Logger.Debug("setting mode to dev")
		lockfile.Mode = lib.Dev
	}

	// TODO: ideal
	// toRemove, err := lib.FindConfigsToRemoval(c)
	// toSymlink, err := lib.FindConfigsToSymlink()
	// toUninstall, err := lib.FindPkgsToUninstall()
	// toInstall, err := lib.FindPkgsToInstall()
	// toUpgrade, err := lib.FindPkgsToUpgrade()
	// diff, err := lockfileBefore.Diff(lockfile)

	if !c.OnlyUninstall && !c.OnlyInstall {
		for _, cfg := range lockfile.Configs {
			if c.CopyMode {
				lib.Logger.Info("copying", "from", cfg.From, "to", cfg.To)
				err := lib.Copy(cfg.From, cfg.To)
				if err != nil {
					lib.Logger.Error("couldn't copy", "err", err)
					return err
				}
			} else {
				lib.Logger.Info("symlinking", "from", cfg.From, "to", cfg.To)
				err := lib.Symlink(cfg.From, cfg.To)
				if err != nil {
					lib.Logger.Error("couldn't symlink", "err", err)
					return err
				}
			}
		}
	} else {
		lib.Logger.Info("skipping copying/symlinking the config, because --only-install or --only-uninstall was passed")
	}

	// previously installed/uninstalled pkgs pointing to idx in the lockfileBefore, so that the information can be copied
	previouslyInstalled := make(map[string]int)
	{
		for idx, cfg := range lockfileBefore.Configs {
			if cfg.InstallInfo.IsInstalled || cfg.InstallInfo.WasUninstalled {
				previouslyInstalled[cfg.Name] = idx
			}
		}

		// copy that installation info from the previous lock regardless of installation/upgrade/uninstallation
		for idx, cfg := range lockfile.Configs {
			if idxInBefore, ok := previouslyInstalled[cfg.Name]; ok {
				prevInstInfo := &lockfileBefore.Configs[idxInBefore].InstallInfo
				lockfile.Configs[idx].InstallInfo = *prevInstInfo
			}
		}
	}

	// config/DEPENDENCIES file parsing
	globalDependencies, err := lib.ParseGlobalDependencies(dirPath)
	if err != nil {
		c.Logger.Error("couldn't parse global dependencies file", "path", dirPath, "err", err)
		return err
	}
	lockfile.GlobalDependencies = globalDependencies
	globalDepsChanged := lib.DidGlobalDependenciesChange(&lockfile.GlobalDependencies, &lockfileBefore.GlobalDependencies)
	globalDepsInstalled := lib.WereGlobalDependenciesInstalled(&lockfileBefore.GlobalDependencies)

	// previously installed/uninstalled global dependencies pointing to idx in the lockfileBefore, so that the information can be copied
	{
		previouslyInstalledGlobal := make(map[string]int)
		for idx, dep := range lockfileBefore.GlobalDependencies {
			if dep.InstallInfo.IsInstalled || dep.InstallInfo.WasUninstalled {
				previouslyInstalledGlobal[dep.Instruction.Pkg] = idx
			}
		}

		// copy that installation info from the previous lock regardless of installation/upgrade/uninstallation
		for idx, dep := range lockfile.GlobalDependencies {
			if idxInBefore, ok := previouslyInstalledGlobal[dep.Instruction.Pkg]; ok {
				prevInstInfo := &lockfileBefore.GlobalDependencies[idxInBefore].InstallInfo
				lockfile.GlobalDependencies[idx].InstallInfo = *prevInstInfo
			}
		}

	}

	if c.Install || c.OnlyInstall || c.Upgrade {
		if globalDepsChanged || !globalDepsInstalled || c.Upgrade {
			err = lib.InstallGlobalDependencies(&lockfile.GlobalDependencies)
			if err != nil {
				lib.Logger.Error("something went wrong while trying to install global dependencies", "err", err)
				return err
			}
		} else {
			lib.Logger.Info("global dependencies didn't change since last installation, not installing", "depsChanged", globalDepsChanged, "previouslyInstalled", globalDepsInstalled)
		}
		for idx, cfg := range lockfile.Configs {
			if _, ok := previouslyInstalled[cfg.Name]; ok {
				instInfo := &cfg.InstallInfo
				if instInfo.IsInstalled && c.Upgrade {
					lib.Logger.Info("upgrading an already installed pkg", "name", cfg.Name)
				} else if instInfo.IsInstalled {
					lib.Logger.Debug("skipping config for installation, because it is already installed, to upgrade pass the --upgrade flag", "name", cfg.Name)
					continue
				} else if instInfo.WasUninstalled {
					lib.Logger.Debug("installing a previously uninstalled pkg", "name", cfg.Name)
				} else {
					panic("shouldnt be possible to get here if its neither installed nor uninstalled")
				}
			}

			if len(pkgs) > 0 && !slices.Contains(pkgs, cfg.Name) {
				lib.Logger.Debug("skipping config for installation, because it wasnt in the provided list", "skipped", cfg.Name)
				continue
			}

			info, err := lib.Install(cfg)
			if err != nil {
				lib.Logger.Error("something went wrong while trying to install using the INSTALL instructions", "pkg", cfg.Name, "err", err)
				continue
			}
			lockfile.Configs[idx].InstallInfo = *info
		}
	}

	namesToIdx := make(map[string]int)
	for idx, cfg := range lockfile.SkippedConfigs {
		namesToIdx[cfg.Name] = idx
	}
	lockDiff := lockfileBefore.Diff(lockfile)
	if c.Uninstall || c.OnlyUninstall {
		for _, _cfg := range lockDiff.NewlySkippedConfigs {
			idx, ok := namesToIdx[_cfg.Name]
			if !ok {
				lib.Logger.Debug("skipping config for uninstallation, because it's not newly skipped", "skipped", _cfg.Name)
				continue
			}
			cfg := lockfile.SkippedConfigs[idx]

			if len(pkgs) > 0 && !slices.Contains(pkgs, cfg.Name) {
				lib.Logger.Debug("skipping config for uninstallation, because it wasnt in the provided list", "skipped", cfg.Name)
				continue
			}

			info, err := lib.Uninstall(cfg)
			if err != nil {
				lib.Logger.Error("something went wrong while trying to uninstall using the instructions", "pkg", cfg.Name, "err", err)
				continue
			}
			lockfile.SkippedConfigs[idx].InstallInfo = *info
		}
	}

	{
		err := lockDiff.Save(lockfileDiffPath, c.DefaultIndent)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save lockfile diff to a file", "err", err)
			return err
		}
		lib.Logger.Info("saved lockfile diff to a file", "path", lockfileDiffPath)

		lib.Logger.Info("removing configs that are no longer in the source")
		err = lib.RemoveConfigsFromTarget(lockDiff.RemovedConfigs)
		if err != nil {
			lib.Logger.Error("something went wrong while removing a config", "mode", lockfile.Mode)
			return err
		}

		lib.Logger.Info("removing configs that are skipped")
		err = lib.RemoveConfigsFromTarget(lockDiff.NewlySkippedConfigs)
		if err != nil {
			lib.Logger.Error("something went wrong while removing a config", "mode", lockfile.Mode)
			return err
		}
	}

	return nil
}
