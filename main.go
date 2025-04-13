package main

import (
	"blanktiger/hm/lib"
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
	beforeLockfile := *lockfile
	logger.Info("lockfile before changes", "lockfile", beforeLockfile)
	defer func() {
		err := lockfile.Save(lockfilePath)
		if err != nil {
			logger.Error("something went wrong while trying to save the lockfile", "err", err)
			return
		}
		logger.Info("lockfile successfully written", "before", beforeLockfile, "after", lockfile)
	}()

	if *dev {
		lockfile.Mode = lib.Dev
	} else {
		lockfile.Mode = lib.Cpy
	}

	// TODO: think if this is correct, for now just reset
	*lockfile = lib.DefaultLockfile

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
		lockfile.AddConfig(from)
	}
}
