package lib

import "encoding/json"
import "os"
import "io"

type Lockfile struct {
	installedConfigs  []Config
	installedPrograms []Program
}

type Config struct {
	name string
}

type Program struct {
	name         string
	requirements []string
}

func readLockfile(txt []byte) (*Lockfile, error) {
	lockfile := Lockfile{}
	err := json.Unmarshal(txt, &lockfile)
	if err != nil {
		return nil, err
	}
	return &lockfile, nil
}

func (l *Lockfile) marshal() ([]byte, error) {
	return json.Marshal(*l)
}

func Copy(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		targetInfo, err := os.Stat(to)
		copyDir(from, to)
	}
}

func copyDir(from, to string) error {
	return nil
}

func CopyFile(from, to string) error {
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
