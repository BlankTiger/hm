package lib

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
		installationInstructions, err := parseInstallationInstructions(path)
		if err != nil {
			return nil, err
		}
		res.Install = *installationInstructions
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

func ExecuteInstall(info requirements) error {
	return nil
}

// func ExecuteUninstall(info Program) error {}

func assert(condition bool, message string) {
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
