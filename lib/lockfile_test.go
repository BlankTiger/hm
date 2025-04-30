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

	diff := DiffLocks(lockBefore, lockAfter)

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

	diff := DiffLocks(lockBefore, lockAfter)

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

	diff := DiffLocks(lockBefore, lockAfter)

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

	diffA := DiffLocks(lockBefore, lockAfterA)
	diffB := DiffLocks(lockBefore, lockAfterB)

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

	diffA := DiffLocks(lockBefore, lockAfterA)
	diffB := DiffLocks(lockBefore, lockAfterB)

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
		UninstallInstructions: []string{},
	},
}

func TestLockfileDiffAddedGlobalDeps(t *testing.T) {
	lockBefore := lockfile{
		GlobalDependencies: []globalDependency{},
	}
	lockAfter := lockfile{
		GlobalDependencies: []globalDependency{commonGlobalDependency},
	}

	diff := DiffLocks(lockBefore, lockAfter)

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

	diff := DiffLocks(lockBefore, lockAfter)

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

func TestUpdateLockfileInstallInfo(t *testing.T) {
	configs := []config{commonCfg}
	lock := lockfile{
		Version:            "",
		Mode:               "",
		GlobalDependencies: []globalDependency{},
		Configs:            configs,
		HiddenConfigs:      []config{},
	}
	time := now()
	expectedInstallInfo := installInfo{
		IsInstalled:           true,
		InstallTime:           time,
		InstallInstruction:    "some nice instruction",
		DependenciesInstalled: false,
		WasUninstalled:        false,
		UninstallTime:         "",
		UninstallInstructions: []string{},
	}
	forUpdate := map[string]installInfo{commonCfg.Name: expectedInstallInfo}

	lock.UpdateInstallInfo(forUpdate)

	assert.Equal(t, expectedInstallInfo, lock.Configs[0].InstallInfo)
}

func TestCopyInstallInfo(t *testing.T) {
	cfgA := createCfg("cargo-binstall")
	cfgB := createCfg("obs-studio")
	cfgC := createCfg("ghostty")
	lockTo := lockfile{
		Configs: []config{cfgA, cfgB, cfgC},
	}

	cfgA.InstallInfo = installInfo{
		IsInstalled:           true,
		InstallTime:           now(),
		InstallInstruction:    "A",
		DependenciesInstalled: true,
		WasUninstalled:        false,
		UninstallTime:         "",
		UninstallInstructions: []string{},
	}
	cfgB.InstallInfo = installInfo{
		IsInstalled:           true,
		InstallTime:           now(),
		InstallInstruction:    "B",
		DependenciesInstalled: true,
		WasUninstalled:        false,
		UninstallTime:         "",
		UninstallInstructions: []string{},
	}
	cfgC.InstallInfo = installInfo{
		IsInstalled:           false,
		InstallTime:           "",
		InstallInstruction:    "",
		DependenciesInstalled: false,
		WasUninstalled:        true,
		UninstallTime:         now(),
		UninstallInstructions: []string{"C"},
	}
	lockFrom := lockfile{
		Configs:       []config{cfgA, cfgB},
		HiddenConfigs: []config{cfgC},
	}

	CopyInstallInfo(&lockFrom, &lockTo)

	expectedConfigsWithInfo := []config{cfgA, cfgB, cfgC}
	assert.Equal(t, lockTo.Configs, expectedConfigsWithInfo)
}
