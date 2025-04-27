# Home Manager (`hm`)

A simple and efficient dotfiles manager for Unix systems that not only syncs
your configuration files but also handles package installation, upgrades and
uninstallation. Created out of frustration with the original `home-manager` nix
project (very slow, probably a skill issue, I know).

## Overview

`hm` (Home Manager) helps you manage your dotfiles and system configurations by:

1. Copying or symlinking configuration files from a source directory to their target locations
2. Installing, uninstalling, and upgrading packages based on instruction files
3. Tracking changes with a lockfile system
4. Managing dependencies for your configurations

## Installation

```bash
# Clone the repository
git clone https://github.com/blanktiger/hm.git

# Build the binary
cd hm

# Make sure ~/.local/bin is in your `$PATH`
go build -o ~/.local/bin/hm
```

## Basic Usage

```bash
# Symlink all configs
hm

# Hard copy all configs
hm --copy

# Install packages along with copying configs
hm --install

# Only install packages without copying configs
hm --only-install

# Upgrade already installed packages
hm --upgrade

# Uninstall packages
hm --uninstall

# Only uninstall packages without modifying configs
hm --only-uninstall

# Manage specific packages
hm --pkgs fish,ghostty
```

## Directory Structure

By default, `hm` looks for configurations in:
- Source directory: `$HOME/.config/homecfg/config/`
- Target directory: `$HOME/.config/`

Each configuration in the source directory should be a folder:

```
$HOME/.config/homecfg/
└── config/
    ├── fish/                 # Fish shell configuration
    │   ├── config.fish
    │   ├── INSTALL           # Installation instructions
    │   └── DEPENDENCIES      # Required dependencies
    ├── nvim/
    │   ├── init.lua
    │   └── INSTALL
    └── .tmux/                # Hidden dirs are skipped (notice the dot)
        └── tmux.conf
```

## Special Files

### INSTALL

The `INSTALL` file contains instructions for installing a package:

```
# Format: method:package
cargo:ripgrep
apt:neovim
pacman:fish
aur:yay
system:firefox
bash:curl -fsSL https://example.com/install.sh | bash
cargo-binstall:bat
```

### DEPENDENCIES

The `DEPENDENCIES` file lists dependencies required for a configuration:

```
cargo:fd
apt:fzf
pacman:git
```

## Advanced Usage

### Using Different Directories

```bash
hm --sourcedir ~/my-dotfiles --targetdir ~/.local
```

### Debug Mode

```bash
hm --dbg
```

## How It Works

1. `hm` scans the source directory for configuration folders
2. For each configuration, it:
   - Copies or symlinks the files to the target location
   - Parses `INSTALL` and `DEPENDENCIES` files if present
   - Installs packages and dependencies when requested
3. Tracks all activities in a lockfile (`hmlock.json`)
4. Generates a diff file (`hmlock_diff.json`) to show changes

## Notes

1. While `hm` supports installing software, it doesn't install tools needed for the installation process itself. For example, `hm` won't install `cargo-binstall` automatically.
2. If installation fails for any package, `hm` will continue processing other configurations.
3. Hidden directories (starting with a dot) are not managed by `hm`.
4. The `.git` directory is always ignored.

## Lockfile

`hm` creates a lockfile (`hmlock.json`) to track:
- What configurations have been deployed
- What packages have been installed
- Installation timestamps
- Skipped configurations
