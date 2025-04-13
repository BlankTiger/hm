package lib

import (
	// "encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// type Lockfile struct {
// 	installedConfigs  []Config
// 	installedPrograms []Program
// }
//
// type Config struct {
// 	name string
// }
//
// type Program struct {
// 	name         string
// 	requirements []string
// }
//
// func readLockfile(txt []byte) (*Lockfile, error) {
// 	lockfile := Lockfile{}
// 	err := json.Unmarshal(txt, &lockfile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &lockfile, nil
// }
//
// func (l *Lockfile) marshal() ([]byte, error) {
// 	return json.Marshal(*l)
// }

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

func copyDir(from, to string) error {
	infoFrom, err := os.Stat(from)
	if err != nil {
		return err
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
