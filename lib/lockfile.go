package lib

import (
	"blanktiger/hm/configuration"
	"encoding/json"
	"io"
	"os"
	"slices"
)

type Mode string

const (
	// hard copy
	Cpy Mode = "copy"
	// symlinks
	Dev Mode = "symlink"
)

type lockfile struct {
	Version            string             `json:"version"`
	Mode               Mode               `json:"mode"`
	GlobalDependencies []globalDependency `json:"globalDependencies"`
	Configs            []config           `json:"configs"`
	HiddenConfigs      []config           `json:"hiddenConfigs"`
}

type globalDependency struct {
	Instruction *installInstruction `json:"installInstruction"`
	InstallInfo installInfo         `json:"installInfo"`
}

func (d *globalDependency) Equal(o *globalDependency) bool {
	instMatch := false
	if d.Instruction != nil && o.Instruction != nil {
		instMatch = *d.Instruction == *o.Instruction
	}

	return instMatch && d.InstallInfo.Equal(&o.InstallInfo)
}

func ContainsGlobalDep(deps []globalDependency, dep globalDependency) bool {
	for _, _dep := range deps {
		if _dep.Equal(&dep) {
			return true
		}
	}
	return false
}

func newGlobalDependency(inst *installInstruction) globalDependency {
	return globalDependency{
		Instruction: inst,
		InstallInfo: installInfo{},
	}
}

func DidGlobalDependenciesChange(depsA, depsB *[]globalDependency) bool {
	namesA := []string{}
	for _, dep := range *depsA {
		namesA = append(namesA, dep.Instruction.Pkg)
	}

	namesB := []string{}
	for _, dep := range *depsB {
		namesB = append(namesB, dep.Instruction.Pkg)
	}

	if len(namesA) != len(namesB) {
		return true
	}

	for _, nameA := range namesA {
		if !slices.Contains(namesB, nameA) {
			return true
		}
	}

	return false
}

func WereGlobalDependenciesInstalled(deps *[]globalDependency) bool {
	for _, dep := range *deps {
		if dep.InstallInfo.IsInstalled || dep.InstallInfo.WasUninstalled {
			return true
		}
	}

	return false
}

func (l *lockfile) AppendSkippedConfig(config config) {
	l.HiddenConfigs = append(l.HiddenConfigs, config)
}

func (l *lockfile) UpdateInstallInfo(info map[string]installInfo) {
	for idx, cfg := range l.Configs {
		if instInfo, ok := info[cfg.Name]; ok {
			l.Configs[idx].InstallInfo = instInfo
		}
	}

	for idx, cfg := range l.HiddenConfigs {
		if instInfo, ok := info[cfg.Name]; ok {
			l.HiddenConfigs[idx].InstallInfo = instInfo
		}
	}
}

func CopyInstallInfo(from, to *lockfile) {
	allSourceCfgs := slices.Concat(from.Configs, from.HiddenConfigs)
configs:
	for _, cfgFrom := range allSourceCfgs {
		for idx := range to.Configs {
			if cfgFrom.Name == to.Configs[idx].Name {
				to.Configs[idx].InstallInfo = cfgFrom.InstallInfo
				continue configs
			}
		}

		for idx := range to.HiddenConfigs {
			if cfgFrom.Name == to.HiddenConfigs[idx].Name {
				to.HiddenConfigs[idx].InstallInfo = cfgFrom.InstallInfo
				continue configs
			}
		}
	}

	for _, depFrom := range from.GlobalDependencies {
		for idx := range to.GlobalDependencies {
			if depFrom.Instruction.Pkg == to.GlobalDependencies[idx].Instruction.Pkg {
				to.GlobalDependencies[idx].InstallInfo = depFrom.InstallInfo
			}
		}
	}
}

func newLockfile() lockfile {
	return lockfile{
		Configs:            []config{},
		HiddenConfigs:      []config{},
		GlobalDependencies: []globalDependency{},
		Mode:               Dev,
		Version:            "0.1.0",
	}
}

var EmptyLockfile = newLockfile()

func CreateLockBasedOnConfigs(c *configuration.Configuration) (*lockfile, error) {
	lockfile := EmptyLockfile
	entries, err := os.ReadDir(c.SourceCfgDir)

	if err != nil {
		Logger.Error("couldn't read dir", "err", err)
		return nil, err
	}

	for _, e := range entries {
		if e.Type() != os.ModeDir {
			continue
		}

		name := e.Name()
		if name == ".git" {
			continue
		}

		from := c.SourceCfgDir + "/" + name
		to := c.TargetDir + "/" + name

		requirements, err := ParseRequirements(from)
		if err != nil {
			Logger.Error("something went wrong while trying to parse requirements", "err", err)
			return nil, err
		}

		if name[0] == '.' {
			Logger.Info("configs", "skipping", name)
			// skipping the dot
			nameIfNotSkipped := name[1:]
			fromIfNotSkipped := c.SourceCfgDir + "/" + nameIfNotSkipped
			toIfNotSkipped := c.TargetDir + "/" + nameIfNotSkipped
			config := NewConfig(nameIfNotSkipped, fromIfNotSkipped, toIfNotSkipped, requirements)
			lockfile.AppendSkippedConfig(config)
			continue
		}

		config := NewConfig(name, from, to, requirements)
		lockfile.AddConfig(config)
	}

	if c.CopyMode {
		Logger.Debug("setting mode to cpy")
		lockfile.Mode = Cpy
	} else {
		Logger.Debug("setting mode to dev")
		lockfile.Mode = Dev
	}

	return &lockfile, nil
}

type lockfileDiff struct {
	AddedConfigs   []config `json:"addedConfigs"`
	RemovedConfigs []config `json:"removedConfigs"`
	// TODO: is this info necessary?
	PreviouslyRemovedConfigs []config           `json:"previouslyRemovedConfigs"`
	AddedGlobalDeps          []globalDependency `json:"addedGlobalDeps"`
	RemovedGlobalDeps        []globalDependency `json:"removedGlobalDeps"`
	ModeChanged              bool               `json:"modeChanged"`
	VersionChanged           bool               `json:"versionChanged"`
}

func (d *lockfileDiff) Save(path, indent string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var toWrite []byte

	if indent == "" {
		toWrite, err = json.Marshal(d)
	} else {
		toWrite, err = json.MarshalIndent(d, "", indent)
	}
	if err != nil {
		return err
	}
	written, err := file.Write(toWrite)
	if err != nil {
		return err
	}
	Assert(written == len(toWrite), "must write what is given")
	return nil
}

func DiffLocks(lockBefore, lockAfter lockfile) lockfileDiff {
	addedConfigs := []config{}
	removedConfigs := []config{}
	newlyHiddenConfigs := []config{}
	previouslyRemovedConfigs := []config{}
	addedGlobalDeps := []globalDependency{}
	removedGlobalDeps := []globalDependency{}

	for _, prevConf := range lockBefore.Configs {
		if !ContainsConfig(lockAfter.Configs, prevConf) {
			removedConfigs = append(removedConfigs, prevConf)
		}

		if ContainsConfig(lockAfter.HiddenConfigs, prevConf) && !ContainsConfig(lockBefore.HiddenConfigs, prevConf) {
			newlyHiddenConfigs = append(newlyHiddenConfigs, prevConf)
		}
	}

	for _, newConf := range lockAfter.Configs {
		if !ContainsConfig(lockBefore.Configs, newConf) {
			addedConfigs = append(addedConfigs, newConf)
		}

	}

	for _, prevSkippedConf := range lockBefore.HiddenConfigs {
		if !ContainsConfig(lockAfter.HiddenConfigs, prevSkippedConf) {
			previouslyRemovedConfigs = append(previouslyRemovedConfigs, prevSkippedConf)
		}
	}

	for _, prevGlobalDep := range lockBefore.GlobalDependencies {
		if !ContainsGlobalDep(lockAfter.GlobalDependencies, prevGlobalDep) {
			removedGlobalDeps = append(removedGlobalDeps, prevGlobalDep)
		}
	}

	for _, newGlobalDep := range lockAfter.GlobalDependencies {
		if !ContainsGlobalDep(lockBefore.GlobalDependencies, newGlobalDep) {
			addedGlobalDeps = append(addedGlobalDeps, newGlobalDep)
		}
	}

	return lockfileDiff{
		AddedConfigs:             addedConfigs,
		RemovedConfigs:           removedConfigs,
		PreviouslyRemovedConfigs: previouslyRemovedConfigs,
		AddedGlobalDeps:          addedGlobalDeps,
		RemovedGlobalDeps:        removedGlobalDeps,
		ModeChanged:              lockBefore.Mode != lockAfter.Mode,
		VersionChanged:           lockBefore.Version != lockAfter.Version,
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
			defaultLockfileBytes, _ := json.Marshal(EmptyLockfile)
			written, err := f.Write(defaultLockfileBytes)
			if err != nil {
				return nil, err
			}
			Assert(written == len(defaultLockfileBytes), "must write what is given")
			return &EmptyLockfile, nil
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

func (l *lockfile) Save(path, indent string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var toWrite []byte
	if indent == "" {
		toWrite, err = json.Marshal(l)
	} else {
		toWrite, err = json.MarshalIndent(l, "", indent)
	}
	if err != nil {
		return err
	}
	written, err := file.Write(toWrite)
	if err != nil {
		return err
	}
	Assert(written == len(toWrite), "must write what is given")
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
