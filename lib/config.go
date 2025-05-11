package lib

import (
	i "blanktiger/hm/instructions"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

type Config struct {
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

	WasUninstalled        bool     `json:"wasUninstalled"`
	UninstallTime         string   `json:"uninstallTime"`
	UninstallInstructions []string `json:"uninstallInstructions"`
}

func (i *installInfo) Equal(o *installInfo) bool {
	uninstInstructionsMatch := false
	if i.UninstallInstructions != nil && o.UninstallInstructions != nil {
		uninstInstructionsMatch = slices.Equal(i.UninstallInstructions, o.UninstallInstructions)
	}
	return uninstInstructionsMatch && i.IsInstalled == o.IsInstalled && i.InstallTime == o.InstallTime && i.InstallInstruction == o.InstallInstruction && i.DependenciesInstalled == o.DependenciesInstalled && i.WasUninstalled == o.WasUninstalled && i.UninstallTime == o.UninstallTime
}

func NewConfig(name, from, to string, reqs *requirements) Config {
	newReqs := newRequirements()
	usedReqs := &newReqs
	if reqs != nil {
		usedReqs = reqs
	}
	return Config{
		Name:         name,
		From:         from,
		To:           to,
		Requirements: *usedReqs,
	}
}

func (c *Config) Equal(o *Config) bool {
	return c.Name == o.Name
}

func ContainsConfig(configs []Config, c Config) bool {
	for _, config := range configs {
		if config.Equal(&c) {
			return true
		}
	}
	return false
}

type requirements struct {
	// TODO: get rid of that Name in the definition of requirements
	// instead we should just pass the config everywhere with all the data
	// (probably)
	Name         string               `json:"name"`
	Install      *installInstruction  `json:"installInstructions"`
	Dependencies []installInstruction `json:"dependencies"`
}

func newRequirements() requirements {
	return requirements{
		Name:         "",
		Install:      nil,
		Dependencies: []installInstruction{},
	}
}

type installInstruction struct {
	Method i.InstallMethod `json:"method"`
	Pkg    string          `json:"pkg"`
}

func newInstallInstruction() installInstruction {
	return installInstruction{
		Method: i.System,
		Pkg:    "",
	}
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
		if txt == "" {
			return nil, nil
		}

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
		Assert(i.IsValidInstallationMethod(method), errMsg)
		res.Method = i.InstallMethod(method)
	}

	{
		pkg := parts[1]
		res.Pkg = strings.Trim(pkg, "\n\t")
	}
	return res, nil
}

func createDepsPath(dir string) string {
	return dir + DEPENDENCIES_PATH_POSTFIX
}

func parseDependencies(path string) (res []installInstruction, err error) {
	res = []installInstruction{}
	path = createDepsPath(path)
	file, err := os.Open(path)
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
