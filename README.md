# Home Manager (`hm`)

A simple and efficient dotfiles manager for Unix systems that not only syncs
your configuration files but also handles package installation, upgrades and
uninstallation. Created out of frustration with the original `home-manager` nix
project (very slow, probably a skill issue, I know).

## Overview

`hm` helps you manage your dotfiles and system configurations by:

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

# Uninstall packages that you prefixed with `.`
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

These directories can be overwritten by doing:

```bash
hm --sourcedir ~/my-dotfiles --targetdir ~/.local
```

Each configuration in the source directory should be a folder:

```
$HOME/.config/homecfg/
└── config/
    ├── DEPENDENCIES          # Global installation instruction file (to install many common dependencies)
    ├── fish/                 # Fish shell configuration
    │   ├── config.fish
    │   ├── INSTALL           # Installation instructions
    │   ├── UNINSTALL         # Uninstallation instructions
    │   └── DEPENDENCIES      # Required dependencies
    ├── nvim/
    │   ├── init.lua
    │   └── INSTALL
    │   ├── UNINSTALL
    └── .tmux/                # Hidden dirs are skipped (notice the dot)
        └── tmux.conf
```

## Special Files

### INSTALL

The `INSTALL` file contains an instruction for installing a package, format is `method:package`. An example if you wanted to install `fish` could be:

```
system:fish
```

Available methods are:

- `system`
- `apt`
- `pacman`
- `dnf`
- `brew`
- `aur`
- `yay`
- `paru`
- `pacaur`
- `aurman`
- `cargo`
- `cargo-binstall`
- `bash`, this executes what you write directly after the `:`. This method doesn't provide automatic uninstall instruction generation, which means that you will not be able to use `--uninstall` to remove a package installed this way.

An example using bash could be:

```
bash:curl -fsSL https://example.com/install.sh | bash
```

The `system` method is particularly powerful as it dynamically detects your
operating system package manager and uses the appropriate installation command.
For example:

- On Debian/Ubuntu systems, it will use `apt`
- On Arch Linux, it will use `pacman`
- On Fedora, it will use `dnf`
- On macOS with Homebrew installed, it will use `brew`

This makes your configuration files more portable across different systems.

Similar thing is implemented for the `aur` method. It will try to find the aur
package manager that is installed.

### UNINSTALL

The `UNINSTALL` file is treated as a bash script. When `INSTALL` contains
packages installed via methods other than `bash`, then this file is redundant,
otherwise you can uninstall the package installed with `bash` method by
specyfing how to do that in this file.

### DEPENDENCIES

The `DEPENDENCIES` file lists dependencies required for a configuration:

```
cargo:fd
system:fzf
system:git
```

### config/DEPENDENCIES

The `config/DEPENDENCIES` file is structurally the same as all the other
`DEPENDENCIES` files in individual config directories. Packages specified in
this file are installed regardless of managed configs before executing
`INSTALL` instructions for individual configs.


## Advanced Usage

### Debug Mode

To show all available logs produced during execution, do:

```bash
hm --dbg
```

## How It Works

1. `hm` scans the source directory for configuration folders
2. Parses `config/DEPENDENCIES` file (if exists) for instructions
   and executes them
3. For each configuration, it:
   - Copies or symlinks the files to the target location
   - Parses `INSTALL` and `DEPENDENCIES` files if present
   - Installs packages and dependencies when requested
4. Tracks all activities in a lockfile (`hmlock.json`)
5. Generates a diff file (`hmlock_diff.json`) to show changes

## Notes

1. While `hm` supports installing software, it doesn't install tools needed for
   the installation process itself. For example, `hm` won't install
   `cargo-binstall` automatically. To do that the recommended way is to put
   `system:cargo-binstall` instruction in the `config/DEPENDENCIES` file.
2. If installation fails for any package, `hm` will continue processing other configurations.
3. Hidden directories (starting with a dot) are not managed by `hm`.
4. The `.git` directory is always ignored.

## Lockfile

`hm` creates a lockfile (`hmlock.json`) in the target directory to track:
- What configurations have been deployed
- What packages have been installed
- Installation timestamps
- Skipped configurations
