package lib

import (
	"blanktiger/hm/instructions"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createCfg(name string) Config {
	return Config{
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
	lockBefore := Lockfile{
		Configs: []Config{commonCfg},
	}
	newCfg := createCfg("zsh")
	lockAfter := Lockfile{
		Configs: []Config{commonCfg, newCfg},
	}

	diff := DiffLocks(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []Config{newCfg},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffRemovedConfigs(t *testing.T) {
	lockBefore := Lockfile{
		Configs: []Config{commonCfg},
	}
	lockAfter := Lockfile{
		Configs: []Config{},
	}

	diff := DiffLocks(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{commonCfg},
		PreviouslyRemovedConfigs: []Config{},
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffPreviouslyRemovedConfigs(t *testing.T) {
	lockBefore := Lockfile{
		HiddenConfigs: []Config{commonCfg},
	}
	lockAfter := Lockfile{
		Configs: []Config{commonCfg},
	}

	diff := DiffLocks(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []Config{commonCfg},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{commonCfg},
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffModeChanged(t *testing.T) {
	lockBefore := Lockfile{
		Mode: Cpy,
	}
	lockAfterA := Lockfile{
		Mode: Dev,
	}
	lockAfterB := Lockfile{
		Mode: Cpy,
	}

	diffA := DiffLocks(lockBefore, lockAfterA)
	diffB := DiffLocks(lockBefore, lockAfterB)

	expectedDiffA := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		ModeChanged:              true,
		VersionChanged:           false,
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
	}
	expectedDiffB := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		ModeChanged:              false,
		VersionChanged:           false,
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
	}
	assert.Equal(t, expectedDiffA, diffA)
	assert.Equal(t, expectedDiffB, diffB)
}

func TestLockfileDiffVersionChanged(t *testing.T) {
	lockBefore := Lockfile{
		Version: "0.1.0",
	}
	lockAfterA := Lockfile{
		Version: "0.1.0",
	}
	lockAfterB := Lockfile{
		Version: "0.2.0",
	}

	diffA := DiffLocks(lockBefore, lockAfterA)
	diffB := DiffLocks(lockBefore, lockAfterB)

	expectedDiffA := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		ModeChanged:              false,
		VersionChanged:           false,
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
	}
	expectedDiffB := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		ModeChanged:              false,
		VersionChanged:           true,
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{},
	}
	assert.Equal(t, expectedDiffA, diffA)
	assert.Equal(t, expectedDiffB, diffB)
}

var commonGlobalDependency = GlobalDependency{
	Instruction: &installInstruction{
		Method: instructions.System,
		Pkg:    "fish",
	},
	InstallInfo: installInfo{
		IsInstalled:           true,
		InstallTime:           now(),
		InstallInstruction:    "sudo pacman -S --noconfirm fish",
		DependenciesInstalled: true,
		WasUninstalled:        false,
		UninstallTime:         "",
		UninstallInstructions: []string{},
	},
}

func TestLockfileDiffAddedGlobalDeps(t *testing.T) {
	lockBefore := Lockfile{
		GlobalDependencies: []GlobalDependency{},
	}
	lockAfter := Lockfile{
		GlobalDependencies: []GlobalDependency{commonGlobalDependency},
	}

	diff := DiffLocks(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		AddedGlobalDeps:          []GlobalDependency{commonGlobalDependency},
		RemovedGlobalDeps:        []GlobalDependency{},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestLockfileDiffRemovedGlobalDeps(t *testing.T) {
	lockBefore := Lockfile{
		GlobalDependencies: []GlobalDependency{commonGlobalDependency},
	}
	lockAfter := Lockfile{
		GlobalDependencies: []GlobalDependency{},
	}

	diff := DiffLocks(lockBefore, lockAfter)

	expectedDiff := lockfileDiff{
		AddedConfigs:             []Config{},
		RemovedConfigs:           []Config{},
		PreviouslyRemovedConfigs: []Config{},
		AddedGlobalDeps:          []GlobalDependency{},
		RemovedGlobalDeps:        []GlobalDependency{commonGlobalDependency},
		ModeChanged:              false,
		VersionChanged:           false,
	}
	assert.Equal(t, expectedDiff, diff)
}

func TestUpdateLockfileInstallInfo(t *testing.T) {
	configs := []Config{commonCfg}
	lock := Lockfile{
		Version:            "",
		Mode:               "",
		GlobalDependencies: []GlobalDependency{},
		Configs:            configs,
		HiddenConfigs:      []Config{},
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
	lockTo := Lockfile{
		Configs: []Config{cfgA, cfgB, cfgC},
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
	lockFrom := Lockfile{
		Configs:       []Config{cfgA, cfgB},
		HiddenConfigs: []Config{cfgC},
	}

	CopyInstallInfo(&lockFrom, &lockTo)

	expectedConfigsWithInfo := []Config{cfgA, cfgB, cfgC}
	assert.Equal(t, lockTo.Configs, expectedConfigsWithInfo)
}
