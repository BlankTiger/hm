package lib

import (
	"blanktiger/hm/instructions"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createCfg(name string) config {
	return config{
		Name: name,
		From: "/some/dir",
		To:   "/some/other/dir",
		Requirements: requirements{
			Name:         name,
			Install:      nil,
			Dependencies: []installInstruction{},
		},
	}
}

var commonCfg = createCfg("fish")

func TestLockfileDiffingAddedConfigs(t *testing.T) {
	lockBefore := lockfile{
		Configs: []config{commonCfg},
	}
	newCfg := createCfg("zsh")
	lockAfter := lockfile{
		Configs: []config{commonCfg, newCfg},
	}

	diff := diffLockfiles(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []config{newCfg},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffRemovedConfigs(t *testing.T) {
	lockBefore := lockfile{
		Configs: []config{commonCfg},
	}
	lockAfter := lockfile{
		Configs: []config{},
	}

	diff := diffLockfiles(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{commonCfg},
		PreviouslyRemovedConfigs: []config{},
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffPreviouslyRemovedConfigs(t *testing.T) {
	lockBefore := lockfile{
		HiddenConfigs: []config{commonCfg},
	}
	lockAfter := lockfile{
		Configs: []config{commonCfg},
	}

	diff := diffLockfiles(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []config{commonCfg},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{commonCfg},
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffModeChanged(t *testing.T) {
	lockBefore := lockfile{
		Mode: Cpy,
	}
	lockAfterA := lockfile{
		Mode: Dev,
	}
	lockAfterB := lockfile{
		Mode: Cpy,
	}

	diffA := diffLockfiles(lockBefore, lockAfterA)
	diffB := diffLockfiles(lockBefore, lockAfterB)

	expectedDiffA := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		ModeChanged:              true,
		VersionChanged:           false,
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
	}
	expectedDiffB := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		ModeChanged:              false,
		VersionChanged:           false,
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
	}
	assert.Equal(t, expectedDiffA, diffA)
	assert.Equal(t, expectedDiffB, diffB)
}

func TestLockfileDiffVersionChanged(t *testing.T) {
	lockBefore := lockfile{
		Version: "0.1.0",
	}
	lockAfterA := lockfile{
		Version: "0.1.0",
	}
	lockAfterB := lockfile{
		Version: "0.2.0",
	}

	diffA := diffLockfiles(lockBefore, lockAfterA)
	diffB := diffLockfiles(lockBefore, lockAfterB)

	expectedDiffA := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		ModeChanged:              false,
		VersionChanged:           false,
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
	}
	expectedDiffB := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		ModeChanged:              false,
		VersionChanged:           true,
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{},
	}
	assert.Equal(t, expectedDiffA, diffA)
	assert.Equal(t, expectedDiffB, diffB)
}

var commonGlobalDependency = globalDependency{
	Instruction: &installInstruction{
		Method: instructions.System,
		Pkg:    "fish",
	},
	InstallInfo: installInfo{
		IsInstalled:           true,
		InstallTime:           now(),
		InstallInstruction:    "sudo pacman -Syy fish",
		DependenciesInstalled: true,
		WasUninstalled:        false,
		UninstallTime:         "",
		UninstallInstruction:  "",
	},
}

func TestLockfileDiffAddedGlobalDeps(t *testing.T) {
	lockBefore := lockfile{
		GlobalDependencies: []globalDependency{},
	}
	lockAfter := lockfile{
		GlobalDependencies: []globalDependency{commonGlobalDependency},
	}

	diff := diffLockfiles(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		AddedGlobalDeps:          []globalDependency{commonGlobalDependency},
		RemovedGlobalDeps:        []globalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffRemovedGlobalDeps(t *testing.T) {
	lockBefore := lockfile{
		GlobalDependencies: []globalDependency{commonGlobalDependency},
	}
	lockAfter := lockfile{
		GlobalDependencies: []globalDependency{},
	}

	diff := diffLockfiles(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []config{},
		RemovedConfigs:           []config{},
		PreviouslyRemovedConfigs: []config{},
		AddedGlobalDeps:          []globalDependency{},
		RemovedGlobalDeps:        []globalDependency{commonGlobalDependency},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}
