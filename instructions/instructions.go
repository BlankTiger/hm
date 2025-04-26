package instructions

import (
	"errors"
	"fmt"
)

type InstallMethod string

const (
	Apt           InstallMethod = "apt"
	Pacman        InstallMethod = "pacman"
	Aur           InstallMethod = "aur"
	Cargo         InstallMethod = "cargo"
	CargoBinstall InstallMethod = "cargo-binstall"
	System        InstallMethod = "system"
	Bash          InstallMethod = "bash"
)

func (i *InstallMethod) IsEmpty() bool {
	return string(*i) == ""
}

func IsValidInstallationMethod(method string) bool {
	switch method {
	case string(Apt), string(Pacman), string(Aur), string(Cargo), string(System), string(Bash), string(CargoBinstall):
		return true
	default:
		return false
	}
}

func (m *InstallMethod) CreateInstallCmd(pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch *m {
	case Cargo:
		cmd, err = installWithCargoCmd(pkg)
	case CargoBinstall:
		cmd, err = installWithCargoBinstallCmd(pkg)
	case System:
		cmd, err = installWithSystemCmd(pkg)
	case Pacman:
		cmd, err = installWithPacmanCmd(pkg)
	case Aur:
		cmd, err = installWithAurCmd(pkg)
	default:
		err = errors.New(fmt.Sprintf("this installation method is either not implemented, or is invalid, method='%s'", *m))
	}

	return cmd, err
}

func (m *InstallMethod) CreateUninstallCmd(pkg string) (cmd string, err error) {
	cmd, err = "", nil

	switch *m {
	case Cargo:
		cmd, err = uninstallWithCargoCmd(pkg)
	case CargoBinstall:
		cmd, err = uninstallWithCargoBinstallCmd(pkg)
	case System:
		cmd, err = uninstallWithSystemCmd(pkg)
	case Pacman:
		cmd, err = uninstallWithPacmanCmd(pkg)
	case Aur:
		cmd, err = uninstallWithAurCmd(pkg)
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
	cmd, err := genSystemInstallCmd(pkg)
	return cmd, err
}

func uninstallWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemUninstallCmd(pkg)
	return cmd, err
}

func genSystemInstallCmd(pkg string) (string, error) {
	panic("not implemented yet")
	cmd := pkg
	return cmd, nil
}

func genSystemUninstallCmd(pkg string) (string, error) {
	panic("not implemented yet")
	cmd := pkg
	return cmd, nil
}
