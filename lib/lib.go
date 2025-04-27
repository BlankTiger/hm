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

var Logger *slog.Logger = nil

const (
	INSTALL_PATH_POSTFIX      = "/INSTALL"
	DEPENDENCIES_PATH_POSTFIX = "/DEPENDENCIES"
)

func ParseGlobalDependencies(path string) (res []globalDependency, err error) {
	res, err = []globalDependency{}, nil

	dependencies, err := parseDependencies(path)
	if err != nil {
		return res, err
	}

	for _, dep := range dependencies {
		res = append(res, newGlobalDependency(&dep))
	}

	return res, err
}

func InstallGlobalDependencies(dependencies *[]globalDependency) error {
	Logger.Info("installing global dependencies")

	for idx, dep := range *dependencies {
		info, err := installGlobalDependency(dep)
		if err != nil {
			return err
		}
		(*dependencies)[idx].InstallInfo = info
	}

	return nil
}

func installGlobalDependency(dep globalDependency) (info installInfo, err error) {
	info, err = installInfo{}, nil

	cmd, err := install(*dep.Instruction)
	if err != nil {
		return info, err
	}

	{
		info.InstallInstruction = cmd
		info.DependenciesInstalled = true
		info.InstallTime = now()
		info.IsInstalled = true
		info.WasUninstalled = false
		info.UninstallInstruction = ""
		info.UninstallTime = ""
	}

	return info, err
}

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
		err = installDependencies(cfg.Requirements.Dependencies)
		if err != nil {
			return res, err
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
		res.InstallTime = now()
		res.IsInstalled = true
		res.WasUninstalled = false
		res.UninstallTime = ""
	}
	return res, err
}

func now() string {
	return time.Now().UTC().Format(time.DateTime)
}

func installDependencies(dependencies []installInstruction) error {
	for _, dep := range dependencies {
		_, err := install(dep)
		if err != nil {
			return err
		}
	}
	return nil
}

func install(inst installInstruction) (cmd string, err error) {
	Assert(!inst.Method.IsEmpty(), fmt.Sprintf("at this point we should always have valid installation instructions, got: '%v'", inst))

	Logger.Info("going to install a pkg", "method", inst.Method, "pkg", inst.Pkg)
	cmd, err = inst.Method.CreateInstallCmd(inst.Pkg)
	Logger.Info("got install cmd", "cmd", cmd)
	if err != nil {
		return cmd, err
	}

	err = execute(cmd)
	return cmd, err
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

	if cfg.Requirements.Install == nil {
		Logger.Debug("not uninstalling the pkg, because there was no INSTALL instructions", "pkg", cfg.Name)
		return res, nil
	}

	// TODO: think on how to correctly handle dependencies when uninstalling, for now dont
	// remove them when uninstalling
	{
		cmd, err := uninstall(*cfg.Requirements.Install)
		if err != nil {
			return res, err
		}
		res.UninstallInstruction = cmd
	}

	// info handling
	{
		res.InstallTime = ""
		res.IsInstalled = false
		res.WasUninstalled = true
		now := time.Now().UTC().Format(time.DateTime)
		res.UninstallTime = now
	}
	return res, err
}

func uninstall(inst installInstruction) (cmd string, err error) {
	Assert(!inst.Method.IsEmpty(), fmt.Sprintf("at this point we should always have valid uninstall instructions, got: '%v'", inst))

	Logger.Info("going to uninstall a pkg", "method", inst.Method, "pkg", inst.Pkg)
	cmd, err = inst.Method.CreateUninstallCmd(inst.Pkg)
	Logger.Info("got uninstall cmd", "cmd", cmd)
	if err != nil {
		return cmd, err
	}

	err = execute(cmd)
	return cmd, err
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
