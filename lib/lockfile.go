package lib

import (
	"encoding/json"
	"io"
	"os"
)

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

func newLockfile() lockfile {
	return lockfile{
		Version:        "0.1.0",
		Configs:        []config{},
		SkippedConfigs: []config{},
	}
}

var DefaultLockfile = newLockfile()

type lockfileDiff struct {
	AddedConfigs             []config `json:"addedConfigs"`
	RemovedConfigs           []config `json:"removedConfigs"`
	NewlySkippedConfigs      []config `json:"newlySkippedConfigs"`
	PreviouslySkippedConfigs []config `json:"previouslySkippedConfigs"`
	ModeChanged              bool     `json:"modeChanged"`
	VersionChanged           bool     `json:"versionChanged"`
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
	assert(written == len(toWrite), "must write what is given")
	return nil
}

// method should be called on an old version of the lockfile
func (l *lockfile) Diff(newLockfile *lockfile) lockfileDiff {
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

	return lockfileDiff{
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
			defaultLockfileBytes, _ := json.Marshal(DefaultLockfile)
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
