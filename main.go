package main

import (
	conf "blanktiger/hm/configuration"
	"blanktiger/hm/instructions"
	"blanktiger/hm/lib"
	"os"
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
	lockBefore, err := lib.ReadOrCreateLockfile(c.LockfilePath)
	if err != nil {
		c.Logger.Info("encountered an error while trying to read an existing lockfile (probably doesnt exist), creating a new one instead", "err", err)
		lockBefore = &lib.EmptyLockfile
	}

	lockAfter, err := lib.CreateLockBasedOnConfigs(c)
	if err != nil {
		c.Logger.Info("something went wrong while trying to create a new lockfile based on your config files in your source directory", "err", err)
		return err
	}

	// config/DEPENDENCIES file parsing
	globalDependencies, err := lib.ParseGlobalDependencies(c.SourceCfgDir)
	if err != nil {
		c.Logger.Error("couldn't parse global dependencies file", "path", c.SourceCfgDir, "err", err)
		return err
	}

	lockAfter.GlobalDependencies = globalDependencies

	lib.CopyInstallInfo(lockBefore, lockAfter)

	diff := lib.DiffLocks(*lockBefore, *lockAfter)

	defer func() {
		err := lockAfter.Save(c.LockfilePath, c.DefaultIndent)
		// NOTE: no explicit return after an error, cause we wanna try to save lockfile diff anyway if we can
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save the lockfile", "err", err)
		}
		err = diff.Save(c.LockfileDiffPath, c.DefaultIndent)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save the lockfile diff", "err", err)
		}
	}()

	globalDepsChanged := lib.DidGlobalDependenciesChange(&lockBefore.GlobalDependencies, &lockAfter.GlobalDependencies)
	globalDepsInstalled := lib.WereGlobalDependenciesInstalled(&lockAfter.GlobalDependencies)
	if c.Install || c.OnlyInstall || c.Upgrade {
		if globalDepsChanged || !globalDepsInstalled || c.Upgrade {
			err = lib.InstallGlobalDependencies(&lockAfter.GlobalDependencies)
			if err != nil {
				lib.Logger.Error("something went wrong while trying to install global dependencies", "err", err)
				return err
			}
		} else {
			lib.Logger.Info("global dependencies didn't change since last installation, not installing", "depsChanged", globalDepsChanged, "previouslyInstalled", globalDepsInstalled)
		}
	}

	if !c.OnlyUninstall && !c.OnlyInstall {
		toSymlink := lockAfter.Configs

		if c.CopyMode {
			err = lib.Copy(c, toSymlink)
		} else {
			err = lib.Symlink(c, toSymlink)
		}
		if err != nil {
			c.Logger.Error("encountered an error while copying/symlinking", "error", err)
			return err
		}

		toRemove := lockAfter.HiddenConfigs
		err = lib.Remove(c, toRemove)
	} else {
		lib.Logger.Info("skipping copying/symlinking the config, because --only-install or --only-uninstall was passed")
	}

	if (c.Install || c.OnlyInstall) || c.Upgrade && !c.OnlyUninstall {
		infoForUpdate := lib.Install(lockAfter)
		lockAfter.UpdateInstallInfo(infoForUpdate)
	}

	if (c.Uninstall || c.OnlyUninstall) && !c.OnlyInstall {
		infoForUpdate := lib.Uninstall(lockAfter)
		lockAfter.UpdateInstallInfo(infoForUpdate)
	}

	return nil
}
