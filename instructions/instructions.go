package instructions

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

var systemPkgManager = INVALID
var Logger *slog.Logger = nil

func FindSystemPkgManager() {
	pkgManager := INVALID

	Logger.Info("Looking for system package manager...")
	if cmdAvailable(Apt, []string{"-v"}) {
		pkgManager = Apt
	} else if cmdAvailable(Pacman, []string{"--version"}) {
		pkgManager = Pacman
	} else if cmdAvailable(Dnf, []string{"-version"}) {
		pkgManager = Dnf
	} else if cmdAvailable(Brew, []string{"--version"}) {
		pkgManager = Brew
	}
	Logger.Info("Result of search for the system package manager", "found", pkgManager)

	systemPkgManager = pkgManager
}

// TODO: implement
// func findAurPkgManager() InstallMethod {}

func cmdAvailable(cmd InstallMethod, args []string) bool {
	command := exec.Command(string(cmd), args...)
	err := command.Run()
	Logger.Debug("error finding cmd", "cmd", cmd, "err", err)

	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode() == 0
	}

	if err != nil {
		return false
	}

	return true
}

type InstallMethod string

const (
	System        InstallMethod = "system"
	Apt           InstallMethod = "apt"
	Pacman        InstallMethod = "pacman"
	Dnf           InstallMethod = "dnf"
	Brew          InstallMethod = "brew"
	Aur           InstallMethod = "aur"
	Cargo         InstallMethod = "cargo"
	CargoBinstall InstallMethod = "cargo-binstall"
	Bash          InstallMethod = "bash"
	INVALID       InstallMethod = ""
)

func (i *InstallMethod) IsEmpty() bool {
	return string(*i) == ""
}

func IsValidInstallationMethod(method string) bool {
	switch method {
	case string(Apt), string(Pacman), string(Dnf), string(Brew), string(Aur), string(Cargo), string(System), string(Bash), string(CargoBinstall):
		return true
	case string(INVALID):
		return false
	default:
		return false
	}
}

func (m *InstallMethod) CreateInstallCmd(pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch *m {
	case System:
		cmd, err = installWithSystemCmd(pkg)
	case Apt:
		cmd, err = installWithAptCmd(pkg)
	case Dnf:
		cmd, err = installWithDnfCmd(pkg)
	case Brew:
		cmd, err = installWithBrewCmd(pkg)
	case Pacman:
		cmd, err = installWithPacmanCmd(pkg)
	case Aur:
		cmd, err = installWithAurCmd(pkg)
	case Cargo:
		cmd, err = installWithCargoCmd(pkg)
	case CargoBinstall:
		cmd, err = installWithCargoBinstallCmd(pkg)
	case Bash:
		// in this case the package is actually a command passed in by the user
		cmd = pkg
	default:
		err = errors.New(fmt.Sprintf("this installation method is either not implemented, or is invalid, method='%s'", *m))
	}

	return cmd, err
}

func (m *InstallMethod) CreateUninstallCmd(pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch *m {
	case System:
		cmd, err = uninstallWithSystemCmd(pkg)
	case Apt:
		cmd, err = uninstallWithAptCmd(pkg)
	case Dnf:
		cmd, err = uninstallWithDnfCmd(pkg)
	case Brew:
		cmd, err = uninstallWithBrewCmd(pkg)
	case Pacman:
		cmd, err = uninstallWithPacmanCmd(pkg)
	case Aur:
		cmd, err = uninstallWithAurCmd(pkg)
	case Cargo:
		cmd, err = uninstallWithCargoCmd(pkg)
	case CargoBinstall:
		cmd, err = uninstallWithCargoBinstallCmd(pkg)
	default:
		err = errors.New(fmt.Sprintf("this uninstallation method is either not implemented, or is invalid, method='%s'", *m))
	}

	return cmd, err
}

func installWithCargoCmd(pkg string) (string, error) {
	cmd := "cargo install " + pkg
	return cmd, nil
}

func uninstallWithCargoCmd(pkg string) (string, error) {
	cmd := "cargo uninstall " + pkg
	return cmd, nil
}

func installWithCargoBinstallCmd(pkg string) (string, error) {
	cmd := "cargo-binstall " + pkg
	return cmd, nil
}

func uninstallWithCargoBinstallCmd(pkg string) (string, error) {
	return uninstallWithCargoCmd(pkg)
}

func installWithPacmanCmd(pkg string) (string, error) {
	cmd := "sudo pacman -Syy " + pkg
	return cmd, nil
}

func uninstallWithPacmanCmd(pkg string) (string, error) {
	cmd := "sudo pacman -R " + pkg
	return cmd, nil
}

func installWithAptCmd(pkg string) (string, error) {
	cmd := "sudo apt install " + pkg
	return cmd, nil
}

func uninstallWithAptCmd(pkg string) (string, error) {
	cmd := "sudo apt remove " + pkg
	return cmd, nil
}

func installWithDnfCmd(pkg string) (string, error) {
	cmd := "dnf install " + pkg
	return cmd, nil
}

func uninstallWithDnfCmd(pkg string) (string, error) {
	cmd := "dnf remove " + pkg
	return cmd, nil
}

func installWithBrewCmd(pkg string) (string, error) {
	cmd := "brew install " + pkg
	return cmd, nil
}

func uninstallWithBrewCmd(pkg string) (string, error) {
	cmd := "brew uninstall " + pkg
	return cmd, nil
}

const aurManager = "yay"

func installWithAurCmd(pkg string) (string, error) {
	// TODO: detect the aur manager used and use that instead of using yay by default
	cmd := aurManager + " -S --sudoloop " + pkg
	return cmd, nil
}

func uninstallWithAurCmd(pkg string) (string, error) {
	cmd := aurManager + " -R " + pkg
	return cmd, nil
}

func installWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemInstallCmd(systemPkgManager, pkg)
	return cmd, err
}

func uninstallWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemUninstallCmd(pkg)
	return cmd, err
}

var (
	couldntFindSysPkgManagerErr = errors.New("couldn't detect system package manager")
	notSystemPkgManagerErr      = errors.New("passed in an installation method that is not a system one")
)

func genSystemInstallCmd(manager InstallMethod, pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch manager {
	case INVALID:
		err = couldntFindSysPkgManagerErr
	default:
		err = notSystemPkgManagerErr
	case Pacman:
		cmd, err = installWithPacmanCmd(pkg)
	case Apt:
		cmd, err = installWithAptCmd(pkg)
	case Dnf:
		cmd, err = installWithDnfCmd(pkg)
	case Brew:
		cmd, err = installWithBrewCmd(pkg)
	}

	return cmd, err
}

func genSystemUninstallCmd(pkg string) (string, error) {
	panic("not implemented yet")
	cmd := pkg
	return cmd, nil
}
