package configuration

import (
	"blanktiger/hm/lib"
	"flag"
	"log/slog"
	"os"
	// "reflect"
)

type Configuration struct {
	// flags
	CopyMode      bool   `help:""`
	Debug         bool   `help:""`
	Install       bool   `help:""`
	OnlyInstall   bool   `help:""`
	Uninstall     bool   `help:""`
	OnlyUninstall bool   `help:""`
	Upgrade       bool   `help:""`
	PkgsTxt       string `help:""`
	SourceDir     string `help:""`
	TargetDir     string `help:""`

	HomeDir       string
	Logger        *slog.Logger
	DefaultIndent string
}

func (c *Configuration) Display() {
	cli_args := "cli args"
	c.Logger.Debug(cli_args, "copy", c.CopyMode)
	c.Logger.Debug(cli_args, "dbg", c.Debug)
	c.Logger.Debug(cli_args, "install", c.Install)
	c.Logger.Debug(cli_args, "only-install", c.OnlyInstall)
	c.Logger.Debug(cli_args, "uninstall", c.Uninstall)
	c.Logger.Debug(cli_args, "only-uninstall", c.OnlyUninstall)
	c.Logger.Debug(cli_args, "upgrade", c.Upgrade)
	c.Logger.Debug(cli_args, "pkgs", c.PkgsTxt)
	c.Logger.Debug(cli_args, "sourcedir", c.SourceDir)
	c.Logger.Debug(cli_args, "targetdir", c.TargetDir)
}

func (c *Configuration) AssertCorrectness() {
	lib.Assert((c.OnlyInstall && c.OnlyUninstall) == false, "cannot pass both --only-install and --only-uninstall")
	lib.Assert((c.Install && c.OnlyUninstall) == false, "cannot pass both --install and --only-uninstall")
	lib.Assert((c.OnlyInstall && c.Uninstall) == false, "cannot pass both --only-install and --uninstall")
	lib.Assert((c.Install && c.Upgrade) == false, "cannot pass both --install and --upgrade flags")
	lib.Assert((c.OnlyInstall && c.Upgrade) == false, "cannot pass both --only-install and --upgrade flags")
	lib.Assert((c.Uninstall && c.Upgrade) == false, "cannot pass both --uninstall and --upgrade flags")
	lib.Assert((c.OnlyUninstall && c.Upgrade) == false, "cannot pass both --only-uninstall and --upgrade flags")
}

func Parse() Configuration {
	var homeDir = os.Getenv("HOME")
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
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		level = slog.LevelDebug
	}

	return Configuration{
		CopyMode:      *copyMode,
		Debug:         *debug,
		Install:       *install,
		OnlyInstall:   *onlyInstall,
		Uninstall:     *uninstall,
		OnlyUninstall: *onlyUninstall,
		Upgrade:       *upgrade,
		PkgsTxt:       *pkgsTxt,
		SourceDir:     *sourcedir,
		TargetDir:     *targetdir,

		HomeDir:       homeDir,
		Logger:        logger,
		DefaultIndent: defaultIndent,
	}
}
