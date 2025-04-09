package main

import "flag"
import "os"
import "log/slog"

// import "blanktiger/hm/lib"

func main() {
	dev := flag.Bool("dev", false, "symlinks the config files, so that changes are instant")
	debug := flag.Bool("dbg", false, "set logging level to debug")
	targetdir := flag.String("targetdir", "~/.config", "target for symlinks for debugging")
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
	logger.Debug(cli_args, "targetdir", *targetdir)

	dirPath := "./config"
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logger.Error("couldn't read dir", "err", err)
		return
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

	}
}
