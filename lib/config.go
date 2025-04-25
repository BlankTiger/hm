package lib

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type config struct {
	Name         string       `json:"name"`
	From         string       `json:"from"`
	To           string       `json:"to"`
	Requirements requirements `json:"requirements"`
	InstallInfo  installInfo  `json:"installInfo"`
}

type installInfo struct {
	IsInstalled bool   `json:"isInstalled"`
	InstallTime string `json:"installTime"`
	// TODO: maybe this is not the right name, maybe should be called InstallCommand?
	InstallInstruction    string `json:"installInstruction"`
	DependenciesInstalled bool   `json:"dependenciesInstalled"`

	WasUninstalled      bool   `json:"wasUninstalled"`
	UninstallTime       string `json:"uninstallTime"`
	UinstallInstruction string `json:"uinstallInstruction"`
}

func NewConfig(name, from, to string, reqs *requirements) config {
	newReqs := newRequirements()
	usedReqs := &newReqs
	if reqs != nil {
		usedReqs = reqs
	}
	return config{
		Name:         name,
		From:         from,
		To:           to,
		Requirements: *usedReqs,
	}
}

func (c *config) Equal(o *config) bool {
	return c.Name == o.Name && c.From == o.From && c.To == o.To
}

func ContainsConfig(configs []config, c config) bool {
	for _, config := range configs {
		if config.Equal(&c) {
			return true
		}
	}
	return false
}

type requirements struct {
	Name         string               `json:"name"`
	Install      *installInstruction  `json:"installInstructions"`
	Uninstall    uninstallInstruction `json:"uninstallInstructions"`
	Dependencies []installInstruction `json:"dependencies"`
}

func newRequirements() requirements {
	return requirements{
		Name:         "",
		Install:      nil,
		Uninstall:    newUninstallInstruction(),
		Dependencies: []installInstruction{},
	}
}

type installInstruction struct {
	Method installMethod `json:"method"`
	Pkg    string        `json:"pkg"`
}

func newInstallInstruction() installInstruction {
	return installInstruction{
		Method: system,
		Pkg:    "",
	}
}

type installMethod string

const (
	apt           = "apt"
	pacman        = "pacman"
	aur           = "aur"
	cargo         = "cargo"
	cargoBinstall = "cargo-binstall"
	system        = "system"
	bash          = "bash"
)

func (i *installMethod) isEmpty() bool {
	return string(*i) == ""
}

func isValidInstallationMethod(method string) bool {
	switch method {
	case apt, pacman, aur, cargo, system, bash, cargoBinstall:
		return true
	default:
		return false
	}
}

// TODO:
// func uninstallationMethodBasedOnInstallationMethod(instMethod installationMethod) UninstallationInstructions

// TODO: think what this should include, for now just type alias to installInstruction
type uninstallInstruction installInstruction

func newUninstallInstruction() uninstallInstruction {
	return uninstallInstruction(newInstallInstruction())
}

func parseInstallInstructions(path string) (res *installInstruction, err error) {
	res = &installInstruction{}
	file, err := os.Open(path + INSTALL_PATH_POSTFIX)
	if err != nil {
		// NOTE: file not existing is not an error in this case (can have config
		// files without installation instructions obviously)
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	{
		txtBytes, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		txt := string(txtBytes)
		res, err = parseInstallInstruction(txt)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func parseInstallInstruction(inst string) (res *installInstruction, err error) {
	newII := newInstallInstruction()
	res = &newII

	// TODO: fix the skip install instruction/commenting install instructions
	if inst[0:2] == "//" {
		Logger.Debug("skipping install instructions, because they are commented out", "instruction", inst)
		return nil, nil
	}

	{
		linesCount := strings.Count(inst, "\n")
		Assert(linesCount <= 1, "think on how to handle multiple installation instructions if we want them in the future")
	}

	parts := strings.Split(inst, ":")
	{
		method := parts[0]
		errMsg := fmt.Sprintf("must be an implemented, valid installation method, instead got: '%s'", method)
		Assert(isValidInstallationMethod(method), errMsg)
		res.Method = installMethod(method)
	}

	{
		pkg := parts[1]
		res.Pkg = strings.Trim(pkg, "\n\t")
	}
	return res, nil
}

func parseUinstallInstructions(path string) (res *uninstallInstruction, err error) {
	panic("unimplemented")
	// return res, err
}

func parseDependencies(path string) (res []installInstruction, err error) {
	res = []installInstruction{}
	file, err := os.Open(path + DEPENDENCIES_PATH_POSTFIX)
	if err != nil {
		// NOTE: file not existing is not an error in this case (can have
		// config files without dependencies obviously)
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, err
	}
	defer file.Close()

	txtBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lines := strings.SplitSeq(string(txtBytes), "\n")
	for line := range lines {
		if line == "" {
			continue
		}
		instructions, err := parseInstallInstruction(line)
		if err != nil {
			return nil, err
		}
		if instructions == nil {
			continue
		}
		res = append(res, *instructions)
	}

	return res, err
}
