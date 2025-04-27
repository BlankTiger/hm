package instructions

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

var Logger *slog.Logger = nil

var systemPkgManager = INVALID

func FindSystemPkgManager() {
	pkgManager := INVALID

	Logger.Info("Looking for system package manager...")
	if cmdAvailable(Apt) {
		pkgManager = Apt
	} else if cmdAvailable(Pacman) {
		pkgManager = Pacman
	} else if cmdAvailable(Dnf) {
		pkgManager = Dnf
	} else if cmdAvailable(Brew) {
		pkgManager = Brew
	}
	Logger.Info("Result of search for the system package manager", "found", pkgManager)

	systemPkgManager = pkgManager
}

var aurPkgManager = INVALID

func FindAurPkgManager() {
	pkgManager := INVALID

	Logger.Info("Looking for aur package manager...")
	if cmdAvailable(Paru) {
		pkgManager = Paru
	} else if cmdAvailable(Yay) {
		pkgManager = Yay
	} else if cmdAvailable(Pacaur) {
		pkgManager = Pacaur
	} else if cmdAvailable(Aurman) {
		pkgManager = Aurman
	}
	Logger.Info("Result of search for the aur package manager", "found", pkgManager)

	aurPkgManager = pkgManager
}

func cmdAvailable(cmd InstallMethod) bool {
	command := exec.Command("which", string(cmd))
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
	System InstallMethod = "system"
	Apt    InstallMethod = "apt"
	Pacman InstallMethod = "pacman"
	Dnf    InstallMethod = "dnf"
	Brew   InstallMethod = "brew"

	Aur    InstallMethod = "aur"
	Yay    InstallMethod = "yay"
	Paru   InstallMethod = "paru"
	Pacaur InstallMethod = "pacaur"
	Aurman InstallMethod = "aurman"

	Cargo         InstallMethod = "cargo"
	CargoBinstall InstallMethod = "cargo-binstall"

	Bash InstallMethod = "bash"

	INVALID InstallMethod = ""
)

func (i *InstallMethod) IsEmpty() bool {
	return string(*i) == ""
}

func IsValidInstallationMethod(method string) bool {
	switch method {

	case string(System), string(Apt), string(Pacman), string(Dnf), string(Brew):
		return true
	case string(Aur), string(Yay), string(Paru), string(Pacaur), string(Aurman):
		return true
	case string(Cargo), string(CargoBinstall), string(Bash):
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

	// system commands
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

	// aur
	case Aur:
		cmd, err = installWithAurCmd(pkg)
	case Yay:
		cmd, err = installWithYayCmd(pkg)
	case Paru:
		cmd, err = installWithParuCmd(pkg)
	case Pacaur:
		cmd, err = installWithPacaurCmd(pkg)
	case Aurman:
		cmd, err = installWithAurmanCmd(pkg)

	// misc
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

	// system commands
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

	// aur
	case Aur:
		cmd, err = uninstallWithAurCmd(pkg)
	case Yay:
		cmd, err = uninstallWithYayCmd(pkg)
	case Paru:
		cmd, err = uninstallWithParuCmd(pkg)
	case Pacaur:
		cmd, err = uninstallWithPacaurCmd(pkg)
	case Aurman:
		cmd, err = uninstallWithAurmanCmd(pkg)

	// misc
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

func installWithAurCmd(pkg string) (string, error) {
	cmd, err := genAurInstallCmd(aurPkgManager, pkg)
	return cmd, err
}

func uninstallWithAurCmd(pkg string) (string, error) {
	cmd, err := genAurUninstallCmd(aurPkgManager, pkg)
	return cmd, err
}

func installWithYayCmd(pkg string) (string, error) {
	cmd := "yay -S --sudoloop " + pkg
	return cmd, nil
}

func uninstallWithYayCmd(pkg string) (string, error) {
	cmd := "yay -R " + pkg
	return cmd, nil
}

func installWithParuCmd(pkg string) (string, error) {
	cmd := "paru -S --sudoloop " + pkg
	return cmd, nil
}

func uninstallWithParuCmd(pkg string) (string, error) {
	cmd := "paru -R " + pkg
	return cmd, nil
}

func installWithPacaurCmd(pkg string) (string, error) {
	// TODO: verify
	cmd := "pacaur -S " + pkg
	return cmd, nil
}

func uninstallWithPacaurCmd(pkg string) (string, error) {
	// TODO: verify
	cmd := "pacaur -R " + pkg
	return cmd, nil
}

func installWithAurmanCmd(pkg string) (string, error) {
	// TODO: verify
	cmd := "aurman -S " + pkg
	return cmd, nil
}

func uninstallWithAurmanCmd(pkg string) (string, error) {
	// TODO: verify
	cmd := "aurman -R " + pkg
	return cmd, nil
}

func installWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemInstallCmd(systemPkgManager, pkg)
	return cmd, err
}

func uninstallWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemUninstallCmd(systemPkgManager, pkg)
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

func genSystemUninstallCmd(manager InstallMethod, pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch manager {

	case INVALID:
		err = couldntFindSysPkgManagerErr
	default:
		err = notSystemPkgManagerErr

	case Pacman:
		cmd, err = uninstallWithPacmanCmd(pkg)
	case Apt:
		cmd, err = uninstallWithAptCmd(pkg)
	case Dnf:
		cmd, err = uninstallWithDnfCmd(pkg)
	case Brew:
		cmd, err = uninstallWithBrewCmd(pkg)

	}

	return cmd, nil
}

var (
	couldntFindAurPkgManagerErr = errors.New("couldn't detect aur package manager")
	notAurPkgManagerErr         = errors.New("passed in an installation method that is not an aur one")
)

func genAurInstallCmd(manager InstallMethod, pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch manager {

	case INVALID:
		err = couldntFindAurPkgManagerErr
	default:
		err = notAurPkgManagerErr

	case Yay:
		cmd, err = installWithYayCmd(pkg)
	case Paru:
		cmd, err = installWithParuCmd(pkg)
	case Pacaur:
		cmd, err = installWithPacaurCmd(pkg)
	case Aurman:
		cmd, err = installWithAurmanCmd(pkg)

	}

	return cmd, err
}

func genAurUninstallCmd(manager InstallMethod, pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch manager {

	case INVALID:
		err = couldntFindAurPkgManagerErr
	default:
		err = notAurPkgManagerErr

	case Yay:
		cmd, err = uninstallWithYayCmd(pkg)
	case Paru:
		cmd, err = uninstallWithParuCmd(pkg)
	case Pacaur:
		cmd, err = uninstallWithPacaurCmd(pkg)
	case Aurman:
		cmd, err = uninstallWithAurmanCmd(pkg)

	}

	return cmd, err
}
