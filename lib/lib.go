package lib

import "encoding/json"

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
