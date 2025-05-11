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

type Lockfile struct {
	Version            string             `json:"version"`
	Mode               Mode               `json:"mode"`
	GlobalDependencies []GlobalDependency `json:"globalDependencies"`
	Configs            []Config           `json:"configs"`
	HiddenConfigs      []Config           `json:"hiddenConfigs"`
}

type GlobalDependency struct {
	Instruction *installInstruction `json:"installInstruction"`
	InstallInfo installInfo         `json:"installInfo"`
}

func (d *GlobalDependency) Equal(o *GlobalDependency) bool {
	instMatch := false
	if d.Instruction != nil && o.Instruction != nil {
		instMatch = *d.Instruction == *o.Instruction
	}

	return instMatch && d.InstallInfo.Equal(&o.InstallInfo)
}

func ContainsGlobalDep(deps []GlobalDependency, dep GlobalDependency) bool {
	for _, _dep := range deps {
		if _dep.Equal(&dep) {
			return true
		}
	}
	return false
}

func newGlobalDependency(inst *installInstruction) GlobalDependency {
	return GlobalDependency{
		Instruction: inst,
		InstallInfo: installInfo{},
	}
}

func DidGlobalDependenciesChange(depsA, depsB *[]GlobalDependency) bool {
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

func WereGlobalDependenciesInstalled(deps *[]GlobalDependency) bool {
	for _, dep := range *deps {
		if dep.InstallInfo.IsInstalled || dep.InstallInfo.WasUninstalled {
			return true
		}
	}

	return false
}

func (l *Lockfile) AppendSkippedConfig(config Config) {
	l.HiddenConfigs = append(l.HiddenConfigs, config)
}

func (l *Lockfile) UpdateInstallInfo(info map[string]installInfo) {
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

func CopyInstallInfo(from, to *Lockfile) {
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


func cfgIsHiddenBasedOnFrom(from string) bool {
	lastSep := strings.LastIndex(from, "/")
	dirName := from[lastSep+1:]
	return dirName[0] == '.'
}

func unhideConfigPath(from string) string {
	lastSep := strings.LastIndex(from, "/")
	dirName := from[lastSep+1:]
	Assert(dirName[0] == '.', "Passed config must be hidden")
	return from[:lastSep+1] + dirName[1:]
}

func hideConfigPath(from string) string {
	lastSep := strings.LastIndex(from, "/")
	dirName := from[lastSep+1:]
	Assert(dirName[0] != '.', "Passed config must not be hidden")
	return from[:lastSep+1] + "." + dirName
}

func newLockfile() Lockfile {
	return Lockfile{
		Configs:            []Config{},
		HiddenConfigs:      []Config{},
		GlobalDependencies: []GlobalDependency{},
		Mode:               Dev,
		Version:            "0.1.0",
	}
}

var EmptyLockfile = newLockfile()

func CreateLockBasedOnConfigs(c *configuration.Configuration) (*Lockfile, error) {
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
			toIfNotSkipped := c.TargetDir + "/" + nameIfNotSkipped
			from := c.SourceCfgDir + "/" + name
			config := NewConfig(nameIfNotSkipped, from, toIfNotSkipped, requirements)
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
	AddedConfigs   []Config `json:"addedConfigs"`
	RemovedConfigs []Config `json:"removedConfigs"`
	// TODO: is this info necessary?
	PreviouslyRemovedConfigs []Config           `json:"previouslyRemovedConfigs"`
	AddedGlobalDeps          []GlobalDependency `json:"addedGlobalDeps"`
	RemovedGlobalDeps        []GlobalDependency `json:"removedGlobalDeps"`
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

func DiffLocks(lockBefore, lockAfter Lockfile) lockfileDiff {
	addedConfigs := []Config{}
	removedConfigs := []Config{}
	newlyHiddenConfigs := []Config{}
	previouslyRemovedConfigs := []Config{}
	addedGlobalDeps := []GlobalDependency{}
	removedGlobalDeps := []GlobalDependency{}

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

func ReadOrCreateLockfile(path string) (*Lockfile, error) {
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

func (l *Lockfile) Save(path, indent string) error {
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

func (l *Lockfile) AddConfig(config Config) {
	l.Configs = append(l.Configs, config)
}

func parseLockfile(txt []byte) (*Lockfile, error) {
	lockfile := Lockfile{}
	err := json.Unmarshal(txt, &lockfile)
	if err != nil {
		return nil, err
	}
	return &lockfile, nil
}
