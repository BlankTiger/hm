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
	InstallInfo  installInfo  `json:"installationInfo"`
}

type installInfo struct {
	IsInstalled             bool   `json:"isInstalled"`
	InstallationTime        string `json:"installationTime"`
	InstallationInstruction string `json:"installationInstruction"`

	WasUninstalled           bool   `json:"wasUninstalled"`
	UninstallationTime       string `json:"uninstallationTime"`
	UinstallationInstruction string `json:"uinstallationInstruction"`
}

func NewConfig(name, from, to string, reqs *requirements) config {
	usedReqs := &requirements{}
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
	Name         string                     `json:"name"`
	Install      installationInstruction    `json:"installationInstructions"`
	Uninstall    uninstallationInstructions `json:"uninstallInstructions"`
	Dependencies []installationInstruction  `json:"dependencies"`
}

type installationInstruction struct {
	Method installationMethod `json:"method"`
	Pkg    string             `json:"pkg"`
}

type installationMethod string

const (
	apt    = "apt"
	pacman = "pacman"
	cargo  = "cargo"
	system = "system"
	bash   = "bash"
)

func isValidInstallationMethod(method string) bool {
	switch method {
	case apt, pacman, cargo, system, bash:
		return true
	default:
		return false
	}
}

// TODO:
// func uninstallationMethodBasedOnInstallationMethod(instMethod installationMethod) UninstallationInstructions

// TODO: think what this should include
type uninstallationInstructions struct{}

func parseInstallationInstructions(path string) (res *installationInstruction, err error) {
	res = &installationInstruction{}
	file, err := os.Open(path + INSTALL_PATH_POSTFIX)
	if err != nil {
		// NOTE: file not existing is not an error in this case (can have config
		// files without installation instructions obviously)
		if os.IsNotExist(err) {
			return res, nil
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
		res, err = parseSingleInstallationInstruction(txt)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func parseSingleInstallationInstruction(inst string) (res *installationInstruction, err error) {
	res = &installationInstruction{}

	{
		linesCount := strings.Count(inst, "\n")
		assert(linesCount <= 1, "think on how to handle multiple installation instructions if we want them in the future")
	}

	parts := strings.Split(inst, ":")
	{
		method := parts[0]
		errMsg := fmt.Sprintf("must be an implemented, valid installation method, instead got: '%s'", method)
		assert(isValidInstallationMethod(method), errMsg)
		res.Method = installationMethod(method)
	}

	{
		pkg := parts[1]
		res.Pkg = strings.Trim(pkg, "\n\t")
	}
	return res, nil
}

func parseUinstallationInstructions(path string) (res *uninstallationInstructions, err error) {
	panic("unimplemented")
	// return res, err
}

func parseDependencies(path string) (res []installationInstruction, err error) {
	res = []installationInstruction{}
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
		instructions, err := parseSingleInstallationInstruction(line)
		if err != nil {
			return nil, err
		}
		res = append(res, *instructions)
	}

	return res, err
}
