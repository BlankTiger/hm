# Home Manager (`hm`)

A simple and efficient dotfiles manager for Unix systems that not only syncs
your configuration files but also handles package installation, upgrades and
uninstallation. Created out of frustration with the original `home-manager` nix
project (very slow, probably a skill issue, I know).

## Overview

`hm` helps you manage your dotfiles and system configurations by:

1. Copying or symlinking configuration files from a source directory to their
   target locations
2. Installing, uninstalling, and upgrading packages based on instruction files
3. Tracking changes with a lockfile system
4. Managing dependencies for your configurations
5. Providing an interactive TUI for easier configuration management

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

# Acts like passing both --install and --uninstall at the same time
hm --manage

# Manage specific packages
hm --pkgs fish,ghostty

# Use the interactive TUI mode
hm --tui

# Enable debug output
hm --dbg
```

## TUI Mode

`hm` includes an interactive Text User Interface (TUI) mode that allows you to:

- Select which configurations to symlink/copy
- Choose which global dependencies to install
- Persist your selections to disk

To use the TUI mode, run:

```bash
hm --tui
```

Navigation:
- Use arrow keys (or j/k) to move up and down
- Press `Space` to toggle selection
- Press `Tab` to move to the next screen
- Press `Shift+Tab` to move to the previous screen
- Press `q` or `Ctrl+c` to exit

The TUI has four screens:
1. Program flags - manage CLI flags interactively
2. Config selection - choose which configs to copy/symlink
3. Global dependencies - select which packages to install
4. Additional options - decide whether to persist your selections

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
    │   ├── UNINSTALL         # Additional shell script run during uninstallation
    │   └── DEPENDENCIES      # Required dependencies
    ├── nvim/
    │   ├── init.lua
    │   ├── INSTALL
    │   └── UNINSTALL
    └── .tmux/                # Hidden dirs are skipped (notice the dot)
        ├── tmux.conf
        ├── INSTALL           # Used to determine uninstallation method for dirs hidden for the first time
        └── UNINSTALL         # Shell script for cleanup (running once when dir is hidden for the first time)
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

Similarly, the `aur` method will detect which AUR helper is installed on your system
(paru, yay, pacaur, or aurman) and use it automatically.

### UNINSTALL

The `UNINSTALL` file is treated as a bash script that will be executed during uninstallation.

When `INSTALL` contains packages installed via methods other than `bash`, the `UNINSTALL` file is
optional since `hm` can generate uninstallation commands automatically.

However, for packages installed with the `bash` method, you need to provide an `UNINSTALL` file
to specify how to remove the package. You can also use this file as a cleanup script, since it
will always run during uninstallation regardless of installation method.

Example `UNINSTALL` file:

```bash
#!/bin/bash
# Clean up configuration files
rm -rf ~/.cache/my-program
# Run the program's uninstaller
/opt/my-program/uninstall.sh
```

### DEPENDENCIES

The `DEPENDENCIES` file lists dependencies required for a configuration:

```
cargo:fd
system:fzf
system:git
```

Each line follows the same `method:package` format as the `INSTALL` file.

NOTE: Currently dependencies are only installed. They aren't uninstalled when
you uninstall the config that owns them if you don't pass the `--uninstall`
flag.

### config/DEPENDENCIES

The `config/DEPENDENCIES` file (at the root of your config directory) specifies global dependencies
that should be installed regardless of which configurations are active. These are installed before
any configuration-specific packages.

This is useful for installing tools that are required by the installation process itself or for
packages that are common dependencies for multiple configurations.

Example:

```
system:git
system:curl
cargo:cargo-binstall
```

## Advanced Usage

### Debug Mode

To show all available logs produced during execution:

```bash
hm --dbg
```

### Managing Specific Packages

To install, uninstall, or upgrade only specific packages:

```bash
hm --pkgs fish,nvim --install
hm --pkgs tmux --uninstall
hm --pkgs fish,nvim --upgrade
```

### Hidden Configurations

Configurations with directories prefixed with a dot (e.g., `.tmux/`) are considered "hidden"
and won't be symlinked/copied or installed by default. This allows you to temporarily disable
configurations without completely removing them.

To uninstall packages associated with hidden configurations:

```bash
hm --uninstall
```

In TUI mode, you can toggle which configurations are active (not hidden) and
choose whether to persist these selections to disk.

## Lockfile System

`hm` creates a lockfile (`hmlock.json`) in the target directory to track:
- What configurations have been deployed
- What packages have been installed
- Installation timestamps
- Hidden configurations
- Global dependencies

The lockfile format is JSON and includes:
- `version`: The version of the lockfile format
- `mode`: Whether configs are symlinked or copied
- `globalDependencies`: List of globally installed packages
- `configs`: List of active configuration directories
- `hiddenConfigs`: List of configuration directories that have been hidden

Additionally, a diff file (`hmlock_diff.json`) is generated to show changes between runs, including:
- Added/removed configurations
- Added/removed global dependencies
- Changes in mode or version

## How It Works

1. `hm` scans the source directory for configuration folders
2. Parses `config/DEPENDENCIES` file (if exists) for global dependencies
   and installs them
3. For each configuration, it:
   - Copies or symlinks the files to the target location
   - Parses `INSTALL`, `DEPENDENCIES` and `UNINSTALL` files if present
   - Installs packages and dependencies when requested
   - Uninstalls packages when requested
4. Tracks all activities in a lockfile (`hmlock.json`)
5. Generates a diff file (`hmlock_diff.json`) to show changes

## Notes

1. While `hm` supports installing software, it doesn't install tools needed for
   the installation process itself. For example, `hm` won't install
   `cargo-binstall` automatically, even if you use `cargo-binstall` as an
   installation method for some package. To do that the recommended way is to put
   `system:cargo-binstall` instruction in the `config/DEPENDENCIES` file.
2. If installation fails for any package, `hm` will continue processing other
   configurations.
3. Hidden directories (starting with a dot) are not managed by `hm` by default.
4. If you hide a configuration directory previously managed by `hm`, you can
   uninstall its dependencies by passing `--uninstall`.
5. The `.git` directory is always ignored.
6. When running in TUI mode, the command-line flags still apply but can be
   modified in the first screen of the interface.
