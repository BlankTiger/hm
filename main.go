package main

import (
	"blanktiger/hm/instructions"
	"blanktiger/hm/lib"
	"flag"
	"log/slog"
	"os"
	"slices"
	"strings"
)

var homeDir = os.Getenv("HOME")

type configuration struct {
	// flags
	copyMode      bool
	debug         bool
	install       bool
	onlyInstall   bool
	uninstall     bool
	onlyUninstall bool
	upgrade       bool
	pkgsTxt       string
	sourceDir     string
	targetDir     string

	logger        *slog.Logger
	defaultIndent string
}

func (c *configuration) display() {
	cli_args := "cli args"
	c.logger.Debug(cli_args, "copy", c.copyMode)
	c.logger.Debug(cli_args, "dbg", c.debug)
	c.logger.Debug(cli_args, "install", c.install)
	c.logger.Debug(cli_args, "only-install", c.onlyInstall)
	c.logger.Debug(cli_args, "uninstall", c.uninstall)
	c.logger.Debug(cli_args, "only-uninstall", c.onlyUninstall)
	c.logger.Debug(cli_args, "upgrade", c.upgrade)
	c.logger.Debug(cli_args, "pkgs", c.pkgsTxt)
	c.logger.Debug(cli_args, "sourcedir", c.sourceDir)
	c.logger.Debug(cli_args, "targetdir", c.targetDir)
}

func (c *configuration) assertCorrectness() {
	lib.Assert((c.onlyInstall && c.onlyUninstall) == false, "cannot pass both --only-install and --only-uninstall")
	lib.Assert((c.install && c.onlyUninstall) == false, "cannot pass both --install and --only-uninstall")
	lib.Assert((c.onlyInstall && c.uninstall) == false, "cannot pass both --only-install and --uninstall")
	lib.Assert((c.install && c.upgrade) == false, "cannot pass both --install and --upgrade flags")
	lib.Assert((c.onlyInstall && c.upgrade) == false, "cannot pass both --only-install and --upgrade flags")
	lib.Assert((c.uninstall && c.upgrade) == false, "cannot pass both --uninstall and --upgrade flags")
	lib.Assert((c.onlyUninstall && c.upgrade) == false, "cannot pass both --only-uninstall and --upgrade flags")

}

func main() {
	c := parseConfigurationFromArgs()
	c.display()
	c.assertCorrectness()

	err := _main(&c)
	if err != nil {
		c.logger.Error("program exited with an error", "error", err)
		os.Exit(1)
	}
}

func parseConfigurationFromArgs() configuration {
	copyMode := flag.Bool("copy", false, "copies the config files instead of symlinking them")
	debug := flag.Bool("dbg", false, "set logging level to debug")

	// TODO: think if this is something that should be done at all times, or not
	// saveLockDiff := flag.Bool("save-diff", false, "wheter to save lockfile diff from before and after to a file regardless of the --debug flag")

	install := flag.Bool("install", false, "whether to install packages using INSTALL instructions found in config folders")
	onlyInstall := flag.Bool("only-install", false, "doesnt copy configs over, only installs the packages that would be copied over based on their INSTALL instructions, --install can be omitted if this option is used")

	uninstall := flag.Bool("uninstall", false, "whether to uninstall packages using INSTALL instructions found in config folders")
	onlyUninstall := flag.Bool("only-uninstall", false, "doesnt copy configs over, only uninstalls the packages for configs that would be removed based on their instructions, --uninstall can be omitted if this option is used")

	upgrade := flag.Bool("upgrade", false, "whether to upgrade already installed packages, for now simply reruns the original install instruction")

	pkgsTxt := flag.String("pkgs", "", "installs/uninstalls only the packages specified by this argument, empty means work on all active, non-hidden configs, example: --pkgs fish,ghostty")

	sourcedir := flag.String("sourcedir", homeDir+"/.config/homecfg", "source of configuration files, without the trailing /")
	// TODO: UNCOMMENT AFTER FINISHING TESTING
	// targetDirDefault := homeDir + "/.config"
	targetDirDefault := homeDir + "/.configbkp"
	targetdir := flag.String("targetdir", targetDirDefault, "target for symlinks for debugging, without the trailing /")
	flag.Parse()

	defaultIndent := "    "
	var level = slog.LevelInfo
	var opts = slog.HandlerOptions{Level: &level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	lib.Logger = logger
	instructions.Logger = logger
	// TODO: make this better, if this gets commented out, then we won't ever
	// find the system package manager
	instructions.FindSystemPkgManager()
	instructions.FindAurPkgManager()
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		level = slog.LevelDebug
	}

	return configuration{
		copyMode:      *copyMode,
		debug:         *debug,
		install:       *install,
		onlyInstall:   *onlyInstall,
		uninstall:     *uninstall,
		onlyUninstall: *onlyUninstall,
		upgrade:       *upgrade,
		pkgsTxt:       *pkgsTxt,
		sourceDir:     *sourcedir,
		targetDir:     *targetdir,

		logger:        logger,
		defaultIndent: defaultIndent,
	}
}

func _main(c *configuration) error {
	dirPath := c.sourceDir + "/config"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		lib.Logger.Error("couldn't read dir", "err", err)
		return err
	}

	lockfilePath := c.targetDir + "/hmlock.json"
	lockfileDiffPath := c.targetDir + "/hmlock_diff.json"
	lockfile, err := lib.ReadOrCreateLockfile(lockfilePath)
	if err != nil {
		lib.Logger.Error("something went wrong while parsing the lockfile, you might need to remove it manually (possibly was generated by previous version of `hm`", "err", err)
		return err
	}
	lockfileBefore := *lockfile
	defer func() {
		err := lockfile.Save(lockfilePath, c.defaultIndent)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save the lockfile", "err", err)
			return
		}
	}()

	// TODO: think if this is correct, for now just reset
	*lockfile = lib.DefaultLockfile

	if c.copyMode {
		lib.Logger.Debug("setting mode to cpy")
		lockfile.Mode = lib.Cpy
	} else {
		lib.Logger.Debug("setting mode to dev")
		lockfile.Mode = lib.Dev
	}

	for _, e := range entries {
		if e.Type() != os.ModeDir {
			continue
		}

		name := e.Name()
		if name == ".git" {
			continue
		}

		from := dirPath + "/" + name
		to := c.targetDir + "/" + name

		if name[0] == '.' {
			lib.Logger.Info("configs", "skipping", name)
			// skipping the dot
			nameIfNotSkipped := name[1:]
			fromIfNotSkipped := dirPath + "/" + nameIfNotSkipped
			toIfNotSkipped := c.targetDir + "/" + nameIfNotSkipped
			config := lib.NewConfig(nameIfNotSkipped, fromIfNotSkipped, toIfNotSkipped, nil)
			lockfile.AppendSkippedConfig(config)
			continue
		}

		requirements, err := lib.ParseRequirements(from)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to parse requirements", "err", err)
			return err
		}

		config := lib.NewConfig(name, from, to, requirements)
		lockfile.AddConfig(config)
	}

	pkgs := []string{}
	if c.pkgsTxt != "" {
		pkgs = strings.Split(c.pkgsTxt, ",")
	}

	if !c.onlyUninstall && !c.onlyInstall {
		for _, cfg := range lockfile.Configs {
			if c.copyMode {
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
		c.logger.Error("couldn't parse global dependencies file", "path", dirPath, "err", err)
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

	if c.install || c.onlyInstall || c.upgrade {
		if globalDepsChanged || !globalDepsInstalled || c.upgrade {
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
				if instInfo.IsInstalled && c.upgrade {
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
	if c.uninstall || c.onlyUninstall {
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
		err := lockDiff.Save(lockfileDiffPath, c.defaultIndent)
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
