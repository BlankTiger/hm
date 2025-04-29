package lib

import (
	"blanktiger/hm/configuration"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindConfigsToSymlink(t *testing.T) {
	c := configuration.Configuration{}
	l := newLockfile()
	l.Configs = append(l.Configs, config{
		Name:         "zsh",
		From:         "abc",
		To:           "bcd",
		Requirements: newRequirements(),
		InstallInfo:  newInstallInfo(),
	})

	toSymlink, err := FindConfigsToSymlink(&c, &l)

	assert.Nil(t, err)
	assert.Equal(t, l.Configs, toSymlink)
}

func TestFindConfigsToRemove(t *testing.T) {
	c := configuration.Configuration{}
	l := newLockfile()
	l.HiddenConfigs = append(l.Configs, config{
		Name:         "zsh",
		From:         "abc",
		To:           "bcd",
		Requirements: newRequirements(),
		InstallInfo:  newInstallInfo(),
	})

	toRemove, err := FindConfigsToRemove(&c, &l)

	assert.Nil(t, err)
	assert.Equal(t, l.HiddenConfigs, toRemove)
}
