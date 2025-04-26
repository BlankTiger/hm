package lib

func installWithCargoCmd(pkg string) (string, error) {
	cmd := "cargo install " + pkg
	return cmd, nil
}

func installWithCargoBinstallCmd(pkg string) (string, error) {
	cmd := "cargo-binstall " + pkg
	return cmd, nil
}

func installWithPacmanCmd(pkg string) (string, error) {
	cmd := "sudo pacman -Syy " + pkg
	return cmd, nil
}

func installWithAurCmd(pkg string) (string, error) {
	// TODO: detect the aur manager used and use that instead of using yay by default
	cmd := "yay -S --sudoloop " + pkg
	return cmd, nil
}

func installWithSystemCmd(pkg string) (string, error) {
	cmd, err := genSystemInstallCmd(pkg)
	return cmd, err
}

func genSystemInstallCmd(pkg string) (string, error) {
	panic("not implemented yet")
	cmd := pkg
	return cmd, nil
}
