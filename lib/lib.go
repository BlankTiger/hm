package lib

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type InstallationMode int

const (
	// symlinks
	Dev InstallationMode = iota
	// hard copy
	Cpy
)

type Lockfile struct {
	Mode              InstallationMode `json:"installationMode"`
	InstalledConfigs  []Config         `json:"installedConfigs"`
	InstalledPrograms []Program        `json:"installedPrograms"`
}

var defaultLockfile = Lockfile{
	InstalledConfigs: []Config{
		{Name: "abcd"},
	},
	InstalledPrograms: []Program{},
}

type Config struct {
	Name string `json:"name"`
}

type Program struct {
	Name         string   `json:"name"`
	Requirements []string `json:"requirements"`
}

func assert(condition bool, message string) {
	if !condition {
		panic(message)
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
			defaultLockfileBytes, _ := defaultLockfile.Marshal()
			written, err := f.Write(defaultLockfileBytes)
			if err != nil {
				return nil, err
			}
			assert(written == len(defaultLockfileBytes), "must write what is given")
			return &defaultLockfile, nil
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

func (l *Lockfile) Save(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
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

func (l *Lockfile) AddConfig(name string) {
	l.InstalledConfigs = append(l.InstalledConfigs, Config{Name: name})
}

func parseLockfile(txt []byte) (*Lockfile, error) {
	lockfile := Lockfile{}
	err := json.Unmarshal(txt, &lockfile)
	if err != nil {
		return nil, err
	}
	return &lockfile, nil
}

func (l *Lockfile) Marshal() ([]byte, error) {
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
