package main

import (
	"blanktiger/hm/lib"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
)

func main() {
	homeDir := os.Getenv("HOME")
	dev := flag.Bool("dev", false, "symlinks the config files, so that changes are instant")
	debug := flag.Bool("dbg", false, "set logging level to debug")
	sourcedir := flag.String("sourcedir", homeDir+"/.config/homecfg", "source of configuration files, without the trailing /")
	// TODO: UNCOMMENT AFTER FINISHING TESTING
	// targetDirDefault := homeDir + "/.config"
	targetDirDefault := homeDir + "/.configbkp"
	targetdir := flag.String("targetdir", targetDirDefault, "target for symlinks for debugging, without the trailing /")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		level = slog.LevelDebug
	}
	opts := slog.HandlerOptions{Level: &level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	cli_args := "cli args"
	logger.Debug(cli_args, "dev", *dev)
	logger.Debug(cli_args, "dbg", *debug)
	logger.Debug(cli_args, "sourcedir", *sourcedir)
	logger.Debug(cli_args, "targetdir", *targetdir)

	dirPath := *sourcedir + "/config"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logger.Error("couldn't read dir", "err", err)
		return
	}

	lockfilePath := *targetdir + "/hmlock.json"
	lockfile, err := lib.ReadOrCreateLockfile(lockfilePath)
	if err != nil {
		logger.Error("something went wrong while parsing the lockfile", "err", err)
		return
	}
	lockfileBefore := *lockfile
	// logger.Info("lockfile before changes", "lockfile", lockfileBefore)
	defer func() {
		err := lockfile.Save(lockfilePath)
		if err != nil {
			logger.Error("something went wrong while trying to save the lockfile", "err", err)
			return
		}
		// logger.Info("lockfile successfully written", "before", lockfileBefore, "after", lockfileAfter)
	}()

	// TODO: think if this is correct, for now just reset
	*lockfile = lib.DefaultLockfile

	if *dev {
		logger.Debug("setting mode to dev", "dev should be", lib.Dev)
		lockfile.Mode = lib.Dev
	} else {
		logger.Debug("setting mode to cpy", "cpy should be", lib.Cpy)
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

		if name[0] == '.' {
			logger.Info("configs", "skipping", name)
			continue
		}

		from := dirPath + "/" + name
		to := *targetdir + "/" + name

		if *dev {
			logger.Info("symlinking", "from", from, "to", to)
			err := lib.Symlink(from, to)
			if err != nil {
				logger.Error("couldn't symlink", "err", err)
				return
			}
		} else {
			logger.Info("copying", "from", from, "to", to)
			err := lib.Copy(from, to)
			if err != nil {
				logger.Error("couldn't copy", "err", err)
				return
			}
		}
		lockfile.AddConfig(lib.Config{Name: name, From: from, To: to})
	}

	{
		lockDiff := lockfileBefore.Diff(lockfile)
		lockDiffJson, err := json.Marshal(&lockDiff)
		if err != nil {
			logger.Error("couldnt marshal lockdiff", "err", err)
			return
		}
		logger.Info("lockfile diff", "diff", lockDiff, "as json", string(lockDiffJson))

		logger.Info("removing configs that are no longer in the source")
		err = lib.RemoveConfigsFromTarget(lockDiff.RemovedConfigs)
		if err != nil {
			logger.Error("something went wrong while removing a config", "mode", lockfile.Mode)
			return
		}
	}
}
