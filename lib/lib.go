package lib

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Mode int

var Level = slog.LevelInfo
var opts = slog.HandlerOptions{Level: &Level}
var Logger = slog.New(slog.NewTextHandler(os.Stdout, &opts))

const (
	INSTALL_PATH_POSTFIX      = "/INSTALL"
	UNINSTLL_PATH_POSTFIX     = "/UNINSTALL"
	DEPENDENCIES_PATH_POSTFIX = "/DEPENDENCIES"
)

func ParseRequirements(path string) (res *requirements, err error) {
	Logger.Debug("parsing requirements", "path", path)
	res = &requirements{}
	{
		Logger.Debug("parsing installation instructions")
		installationInstructions, err := parseInstallInstructions(path)
		if err != nil {
			return nil, err
		}
		if installationInstructions != nil {
			res.Install = installationInstructions
		}
	}

	{
		// TODO: implement
		// Logger.Debug("parsing uninstallation instructions")
		// uninstallationInstructions, err := parseUinstallationInstructions(path)
	}

	{
		Logger.Debug("parsing dependencies")
		dependencies, err := parseDependencies(path)
		if err != nil {
			return nil, err
		}
		res.Dependencies = dependencies
	}

	return res, err
}

func Install(cfg config) (res *installInfo, err error) {
	res, err = &cfg.InstallInfo, nil

	if cfg.Requirements.Install == nil {
		Logger.Debug("not installing the pkg, because there was no INSTALL instructions", "pkg", cfg.Name)
		return res, nil
	}

	// first install the dependencies if any
	{
		for _, depInst := range cfg.Requirements.Dependencies {
			_, err := install(depInst)
			if err != nil {
				return res, err
			}
		}
		res.DependenciesInstalled = true
	}

	{
		cmd, err := install(*cfg.Requirements.Install)
		if err != nil {
			return res, err
		}
		res.InstallInstruction = cmd
	}

	// info handling
	{
		now := time.Now().UTC().Format(time.DateTime)
		res.InstallTime = now
		res.IsInstalled = true
		res.WasUninstalled = false
		res.UninstallTime = ""
	}
	return res, err
}

func install(inst installInstruction) (cmd string, err error) {
	Assert(!inst.Method.isEmpty(), fmt.Sprintf("at this point we should always have valid installation instructions, got: '%v'", inst))

	cmd, err = "", nil

	Logger.Info("going to install a pkg", "method", inst.Method, "pkg", inst.Pkg)
	switch inst.Method {
	case cargo:
		cmd, err = installWithCargoCmd(inst.Pkg)
	case cargoBinstall:
		cmd, err = installWithCargoBinstallCmd(inst.Pkg)
	case system:
		cmd, err = installWithSystemCmd(inst.Pkg)
	case pacman:
		cmd, err = installWithPacmanCmd(inst.Pkg)
	case aur:
		cmd, err = installWithAurCmd(inst.Pkg)
	default:
		err = errors.New(fmt.Sprintf("this installation method is either not implemented, or is invalid, method='%s'", inst.Method))
	}
	Logger.Info("got install cmd", "cmd", cmd)
	if err != nil {
		return cmd, err
	}

	err = execute(cmd)
	return cmd, err
}

func installWithCargoCmd(pkg string) (string, error) {
	cmd := "cargo install " + pkg
	return cmd, nil
}

func installWithCargoBinstallCmd(pkg string) (string, error) {
	cmd := "cargo-binstall " + pkg
	return cmd, nil
}

func installWithPacmanCmd(pkg string) (string, error) {
	cmd := "sudo pacman -Syy " + pkg
	return cmd, nil
}

func installWithAurCmd(pkg string) (string, error) {
	// TODO: detect the aur manager used and use that instead of using yay by default
	cmd := "yay -S --sudoloop " + pkg
	return cmd, nil
}

func installWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemInstallCmd(pkg)
	return cmd, err
}

func genSystemInstallCmd(pkg string) (string, error) {
	panic("not implemented yet")
	cmd := pkg
	return cmd, nil
}

func execute(cmd string) error {
	splitCmd := strings.Split(cmd, " ")
	{
		execCmd := exec.Command(splitCmd[0], splitCmd[1:]...)
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		// BUG: if user does C-c here, then stdin/stdout/stderr might not get released
		err := execCmd.Start()
		if err != nil {
			return nil
		}
		err = execCmd.Wait()
		if err != nil {
			return err
		}
	}

	Logger.Info("Successfully installed", "cmd", cmd)
	return nil
}

func Uninstall(cfg config) (res *installInfo, err error) {
	res, err = &cfg.InstallInfo, nil
	return res, err
}

func Assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func RemoveConfigsFromTarget(configs []config) error {
	for _, c := range configs {
		Logger.Info("removing config from target", "target", c.To)
		err := os.RemoveAll(c.To)
		if err != nil {
			return err
		}
	}
	return nil
}

func Symlink(from, to string) error {
	_, err := os.Stat(from)
	if err != nil {
		return err
	}
	_, err = os.Stat(to)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil
		}
	}

	err = os.Symlink(from, to)
	if err != nil {
		if os.IsExist(err) {
			err := os.RemoveAll(to)
			if err != nil {
				return err
			}
			err = os.Symlink(from, to)
		} else {
			return err
		}
	}
	return nil
}

func Copy(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		err = copyDir(from, to)
	} else {
		err = copyFile(from, to)
	}
	return err
}

// if file doesn't exist then that is still considered as not a symlink (and no error)
func isSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.Mode()&os.ModeSymlink != 0, nil
}

func copyDir(from, to string) error {
	infoFrom, err := os.Stat(from)
	if err != nil {
		return err
	}
	{
		link, err := isSymlink(to)
		if err != nil {
			return err
		}
		if link {
			err = os.Remove(to)
			if err != nil {
				return err
			}
		}
	}
	infoTo, err := os.Stat(to)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(to, infoFrom.Mode())
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	infoTo, err = os.Stat(to)
	if err != nil {
		return err
	}

	if !infoFrom.IsDir() {
		return errors.New("from is not a directory")
	}
	if !infoTo.IsDir() {
		return errors.New("to is not a directory")
	}

	entries, err := os.ReadDir(from)
	if err != nil {
		return err
	}

	for _, e := range entries {
		name := e.Name()
		fromPath := filepath.Join(from, name)
		toPath := filepath.Join(to, name)

		if e.IsDir() {
			err = copyDir(fromPath, toPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(fromPath, toPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(from, to string) error {
	inputFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	return err
}
