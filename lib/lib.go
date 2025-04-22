package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Mode int

var Level = slog.LevelInfo
var opts = slog.HandlerOptions{Level: &Level}
var Logger = slog.New(slog.NewTextHandler(os.Stdout, &opts))

const (
	// hard copy
	Cpy Mode = iota
	// symlinks
	Dev
)

type lockfile struct {
	Version        string   `json:"version"`
	Mode           Mode     `json:"mode"`
	Configs        []config `json:"configs"`
	SkippedConfigs []config `json:"skippedConfigs"`
}

func (l *lockfile) AppendSkippedConfig(config config) {
	l.SkippedConfigs = append(l.SkippedConfigs, config)
}

func NewLockfile() lockfile {
	return lockfile{
		Version:        "0.1.0",
		Configs:        []config{},
		SkippedConfigs: []config{},
	}
}

var DefaultLockfile = NewLockfile()

type config struct {
	Name         string       `json:"name"`
	From         string       `json:"from"`
	To           string       `json:"to"`
	Requirements requirements `json:"requirements"`
	InstallInfo  installInfo  `json:"installationInfo"`
}

type installInfo struct {
	IsInstalled      bool   `json:"isInstalled"`
	InstallationTime string `json:"installationTime"`
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

type InstallationMethod string

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
// func uninstallationMethodBasedOnInstallationMethod(instMethod InstallationMethod) UninstallationInstructions

const (
	INSTALL_PATH_POSTFIX      = "/INSTALL"
	UNINSTALL_PATH_POSTFIX    = "/UNINSTALL"
	DEPENDENCIES_PATH_POSTFIX = "/DEPENDENCIES"
)

type requirements struct {
	Name         string                     `json:"name"`
	Install      InstallationInstruction    `json:"installationInstructions"`
	Uninstall    UninstallationInstructions `json:"uninstallInstructions"`
	Dependencies []InstallationInstruction  `json:"dependencies"`
}

type InstallationInstruction struct {
	Method InstallationMethod `json:"method"`
	Pkg    string             `json:"pkg"`
}

// TODO: think what this should include
type UninstallationInstructions struct{}

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

func parseInstallationInstructions(path string) (res *InstallationInstruction, err error) {
	res = &InstallationInstruction{}
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

func parseSingleInstallationInstruction(inst string) (res *InstallationInstruction, err error) {
	res = &InstallationInstruction{}

	{
		linesCount := strings.Count(inst, "\n")
		assert(linesCount <= 1, "think on how to handle multiple installation instructions if we want them in the future")
	}

	parts := strings.Split(inst, ":")
	{
		method := parts[0]
		errMsg := fmt.Sprintf("must be an implemented, valid installation method, instead got: '%s'", method)
		assert(isValidInstallationMethod(method), errMsg)
		res.Method = InstallationMethod(method)
	}

	{
		pkg := parts[1]
		res.Pkg = strings.Trim(pkg, "\n\t")
	}
	return res, nil
}

func parseUinstallationInstructions(path string) (res *UninstallationInstructions, err error) {
	panic("unimplemented")
	// return res, err
}

func parseDependencies(path string) (res []InstallationInstruction, err error) {
	res = []InstallationInstruction{}
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

type LockfileDiff struct {
	AddedConfigs             []config `json:"addedConfigs"`
	RemovedConfigs           []config `json:"removedConfigs"`
	NewlySkippedConfigs      []config `json:"newlySkippedConfigs"`
	PreviouslySkippedConfigs []config `json:"previouslySkippedConfigs"`
	ModeChanged              bool     `json:"modeChanged"`
	VersionChanged           bool     `json:"versionChanged"`
}

// method should be called on an old version of the lockfile
func (l *lockfile) Diff(newLockfile *lockfile) LockfileDiff {
	addedConfigs := []config{}
	removedConfigs := []config{}
	newlySkippedConfigs := []config{}
	previouslySkippedConfigs := []config{}

	for _, prevConf := range l.Configs {
		if !ContainsConfig(newLockfile.Configs, prevConf) {
			removedConfigs = append(removedConfigs, prevConf)
		}

		Logger.Debug("DIFFING", "SkippedConfigs", newLockfile.SkippedConfigs, "prevConf", prevConf)
		if ContainsConfig(newLockfile.SkippedConfigs, prevConf) && !ContainsConfig(l.SkippedConfigs, prevConf) {
			newlySkippedConfigs = append(newlySkippedConfigs, prevConf)
		}
	}

	for _, newConf := range newLockfile.Configs {
		if !ContainsConfig(l.Configs, newConf) {
			addedConfigs = append(addedConfigs, newConf)
		}

	}

	for _, prevSkippedConf := range l.SkippedConfigs {
		if !ContainsConfig(newLockfile.SkippedConfigs, prevSkippedConf) {
			previouslySkippedConfigs = append(previouslySkippedConfigs, prevSkippedConf)
		}
	}

	return LockfileDiff{
		AddedConfigs:             addedConfigs,
		RemovedConfigs:           removedConfigs,
		NewlySkippedConfigs:      newlySkippedConfigs,
		PreviouslySkippedConfigs: previouslySkippedConfigs,
		ModeChanged:              l.Mode != newLockfile.Mode,
		VersionChanged:           l.Version != newLockfile.Version,
	}
}

func ReadOrCreateLockfile(path string) (*lockfile, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(path)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			defaultLockfileBytes, _ := DefaultLockfile.Marshal()
			written, err := f.Write(defaultLockfileBytes)
			if err != nil {
				return nil, err
			}
			assert(written == len(defaultLockfileBytes), "must write what is given")
			return &DefaultLockfile, nil
		}
		return nil, err
	}
	defer file.Close()
	txt, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return parseLockfile(txt)
}

func (l *lockfile) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	toWrite, err := l.Marshal()
	if err != nil {
		return err
	}
	written, err := file.Write(toWrite)
	if err != nil {
		return err
	}
	assert(written == len(toWrite), "must write what is given")
	return nil
}

func (l *lockfile) AddConfig(config config) {
	l.Configs = append(l.Configs, config)
}

func parseLockfile(txt []byte) (*lockfile, error) {
	lockfile := lockfile{}
	err := json.Unmarshal(txt, &lockfile)
	if err != nil {
		return nil, err
	}
	return &lockfile, nil
}

func (l *lockfile) Marshal() ([]byte, error) {
	return json.Marshal(*l)
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
