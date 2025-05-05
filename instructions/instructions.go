package instructions

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

var Logger *slog.Logger = nil

var systemPkgManager = INVALID

func Init(l *slog.Logger) {
	Logger = l
	FindSystemPkgManager()
	FindAurPkgManager()
}

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
		cmd = installWithAptCmd(pkg)
	case Dnf:
		cmd = installWithDnfCmd(pkg)
	case Brew:
		cmd = installWithBrewCmd(pkg)
	case Pacman:
		cmd = installWithPacmanCmd(pkg)

	// aur
	case Aur:
		cmd, err = installWithAurCmd(pkg)
	case Yay:
		cmd = installWithYayCmd(pkg)
	case Paru:
		cmd = installWithParuCmd(pkg)
	case Pacaur:
		cmd = installWithPacaurCmd(pkg)
	case Aurman:
		cmd = installWithAurmanCmd(pkg)

	// misc
	case Cargo:
		cmd = installWithCargoCmd(pkg)
	case CargoBinstall:
		cmd = installWithCargoBinstallCmd(pkg)
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
		cmd = uninstallWithAptCmd(pkg)
	case Dnf:
		cmd = uninstallWithDnfCmd(pkg)
	case Brew:
		cmd = uninstallWithBrewCmd(pkg)
	case Pacman:
		cmd = uninstallWithPacmanCmd(pkg)

	// aur
	case Aur:
		cmd, err = uninstallWithAurCmd(pkg)
	case Yay:
		cmd = uninstallWithYayCmd(pkg)
	case Paru:
		cmd = uninstallWithParuCmd(pkg)
	case Pacaur:
		cmd = uninstallWithPacaurCmd(pkg)
	case Aurman:
		cmd = uninstallWithAurmanCmd(pkg)

	// misc
	case Cargo:
		cmd = uninstallWithCargoCmd(pkg)
	case CargoBinstall:
		cmd = uninstallWithCargoBinstallCmd(pkg)

	default:
		err = errors.New(fmt.Sprintf("this uninstallation method is either not implemented, or is invalid, method='%s'", *m))
	}

	return cmd, err
}

func installWithCargoCmd(pkg string) string {
	return "cargo install " + pkg
}

func uninstallWithCargoCmd(pkg string) string {
	return "cargo uninstall " + pkg
}

func installWithCargoBinstallCmd(pkg string) string {
	return "cargo-binstall " + pkg
}

func uninstallWithCargoBinstallCmd(pkg string) string {
	return uninstallWithCargoCmd(pkg)
}

func installWithPacmanCmd(pkg string) string {
	return "sudo pacman -S --noconfirm " + pkg
}

func uninstallWithPacmanCmd(pkg string) string {
	return "sudo pacman -R --noconfirm " + pkg
}

func installWithAptCmd(pkg string) string {
	return "sudo apt install -y " + pkg
}

func uninstallWithAptCmd(pkg string) string {
	return "sudo apt remove -y " + pkg
}

func installWithDnfCmd(pkg string) string {
	return "dnf install " + pkg
}

func uninstallWithDnfCmd(pkg string) string {
	return "dnf remove " + pkg
}

func installWithBrewCmd(pkg string) string {
	return "brew install " + pkg
}

func uninstallWithBrewCmd(pkg string) string {
	return "brew uninstall " + pkg
}

func installWithAurCmd(pkg string) (string, error) {
	return genAurInstallCmd(aurPkgManager, pkg)
}

func uninstallWithAurCmd(pkg string) (string, error) {
	return genAurUninstallCmd(aurPkgManager, pkg)
}

func installWithYayCmd(pkg string) string {
	return "yay -S --sudoloop " + pkg
}

func uninstallWithYayCmd(pkg string) string {
	return "yay -R " + pkg
}

func installWithParuCmd(pkg string) string {
	return "paru -S --sudoloop " + pkg
}

func uninstallWithParuCmd(pkg string) string {
	return "paru -R " + pkg
}

func installWithPacaurCmd(pkg string) string {
	// TODO: verify
	panic("verify")
	return "pacaur -S " + pkg
}

func uninstallWithPacaurCmd(pkg string) string {
	// TODO: verify
	panic("verify")
	return "pacaur -R " + pkg
}

func installWithAurmanCmd(pkg string) string {
	// TODO: verify
	panic("verify")
	return "aurman -S " + pkg
}

func uninstallWithAurmanCmd(pkg string) string {
	// TODO: verify
	panic("verify")
	return "aurman -R " + pkg
}

func installWithSystemCmd(pkg string) (string, error) {
	return genSystemInstallCmd(systemPkgManager, pkg)
}

func uninstallWithSystemCmd(pkg string) (string, error) {
	return genSystemUninstallCmd(systemPkgManager, pkg)
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
		cmd = installWithPacmanCmd(pkg)
	case Apt:
		cmd = installWithAptCmd(pkg)
	case Dnf:
		cmd = installWithDnfCmd(pkg)
	case Brew:
		cmd = installWithBrewCmd(pkg)

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
		cmd = uninstallWithPacmanCmd(pkg)
	case Apt:
		cmd = uninstallWithAptCmd(pkg)
	case Dnf:
		cmd = uninstallWithDnfCmd(pkg)
	case Brew:
		cmd = uninstallWithBrewCmd(pkg)

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
		cmd = installWithYayCmd(pkg)
	case Paru:
		cmd = installWithParuCmd(pkg)
	case Pacaur:
		cmd = installWithPacaurCmd(pkg)
	case Aurman:
		cmd = installWithAurmanCmd(pkg)

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
		cmd = uninstallWithYayCmd(pkg)
	case Paru:
		cmd = uninstallWithParuCmd(pkg)
	case Pacaur:
		cmd = uninstallWithPacaurCmd(pkg)
	case Aurman:
		cmd = uninstallWithAurmanCmd(pkg)

	}

	return cmd, err
}
