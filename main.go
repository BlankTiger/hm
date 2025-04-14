package main

import (
	"blanktiger/hm/lib"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
)

var homeDir = os.Getenv("HOME")

func main() {
	dev := flag.Bool("dev", false, "symlinks the config files, so that changes are instant")
	debug := flag.Bool("dbg", false, "set logging level to debug")
	sourcedir := flag.String("sourcedir", homeDir+"/.config/homecfg", "source of configuration files, without the trailing /")
	// TODO: UNCOMMENT AFTER FINISHING TESTING
	// targetDirDefault := homeDir + "/.config"
	targetDirDefault := homeDir + "/.configbkp"
	targetdir := flag.String("targetdir", targetDirDefault, "target for symlinks for debugging, without the trailing /")
	flag.Parse()

	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		lib.Level = slog.LevelDebug
	}
	cli_args := "cli args"
	lib.Logger.Debug(cli_args, "dev", *dev)
	lib.Logger.Debug(cli_args, "dbg", *debug)
	lib.Logger.Debug(cli_args, "sourcedir", *sourcedir)
	lib.Logger.Debug(cli_args, "targetdir", *targetdir)

	dirPath := *sourcedir + "/config"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		lib.Logger.Error("couldn't read dir", "err", err)
		return
	}

	lockfilePath := *targetdir + "/hmlock.json"
	lockfile, err := lib.ReadOrCreateLockfile(lockfilePath)
	if err != nil {
		lib.Logger.Error("something went wrong while parsing the lockfile", "err", err)
		return
	}
	lockfileBefore := *lockfile
	// lib.Logger.Info("lockfile before changes", "lockfile", lockfileBefore)
	defer func() {
		err := lockfile.Save(lockfilePath)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to save the lockfile", "err", err)
			return
		}
		// lib.Logger.Info("lockfile successfully written", "before", lockfileBefore, "after", lockfileAfter)
	}()

	// TODO: think if this is correct, for now just reset
	*lockfile = lib.DefaultLockfile

	if *dev {
		lib.Logger.Debug("setting mode to dev", "dev should be", lib.Dev)
		lockfile.Mode = lib.Dev
	} else {
		lib.Logger.Debug("setting mode to cpy", "cpy should be", lib.Cpy)
		lockfile.Mode = lib.Cpy
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
		to := *targetdir + "/" + name

		if name[0] == '.' {
			lib.Logger.Info("configs", "skipping", name)
			// skipping the dot
			nameIfNotSkipped := name[1:]
			fromIfNotSkipped := dirPath + "/" + nameIfNotSkipped
			toIfNotSkipped := *targetdir + "/" + nameIfNotSkipped
			config := lib.NewConfig(nameIfNotSkipped, fromIfNotSkipped, toIfNotSkipped, nil)
			lockfile.AppendSkippedConfig(config)
			continue
		}

		requirements, err := lib.ParseRequirements(from)
		if err != nil {
			lib.Logger.Error("something went wrong while trying to parse requirements", "err", err)
			return
		}

		if *dev {
			lib.Logger.Info("symlinking", "from", from, "to", to)
			err := lib.Symlink(from, to)
			if err != nil {
				lib.Logger.Error("couldn't symlink", "err", err)
				return
			}
		} else {
			lib.Logger.Info("copying", "from", from, "to", to)
			err := lib.Copy(from, to)
			if err != nil {
				lib.Logger.Error("couldn't copy", "err", err)
				return
			}
		}

		config := lib.NewConfig(name, from, to, &requirements)
		lockfile.AddConfig(config)
	}

	lockDiff := lockfileBefore.Diff(lockfile)
	{
		lockDiffJson, err := json.Marshal(&lockDiff)
		if err != nil {
			lib.Logger.Error("couldnt marshal lockdiff", "err", err)
			return
		}
		lib.Logger.Info("lockfile diff", "diff", lockDiff, "as json", string(lockDiffJson))

		lib.Logger.Info("removing configs that are no longer in the source")
		err = lib.RemoveConfigsFromTarget(lockDiff.RemovedConfigs)
		if err != nil {
			lib.Logger.Error("something went wrong while removing a config", "mode", lockfile.Mode)
			return
		}

		lib.Logger.Info("removing configs that are skipped")
		err = lib.RemoveConfigsFromTarget(lockDiff.NewlySkippedConfigs)
		if err != nil {
			lib.Logger.Error("something went wrong while removing a config", "mode", lockfile.Mode)
			return
		}
	}

}
