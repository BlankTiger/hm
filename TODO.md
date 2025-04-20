Done:
- basic copying
- symlinking of directories (dev mode) for when I don't want to run `hm` everytime I'm incrementally
  working on some config
- hidden directories are not managed by `hm`
- ignore `.git`
- lockfiles
    + persisting info on what was copied
- diffing for lockfiles

Bugs:
- in case of systemd, if I want to store the enabled targets, then they are always stored as
  symlinks, currently this obviously only works in non `dev` mode, because of that maybe it's smart
  not to store systemd config at first (removing the directory when copying would also reset enabled
  targets because of this)

TODO:

has to depend on the lockfile (we have to store information on how the package was installed)
- move functionality from the main loop to lib.go
- lockfile, need to probably implement some kind of hashing to not overwrite files if there are some
  new ones in directories that are managed with `hm` (this shouldn't happen though, because everything
  should be managed via `hm`)
    + info on what has changed
    + info on what was installed
- installing with instructions from INSTALL
    + installing dependencies from REQUIREMENTS files
    + parsing REQUIREMENTS -> some requirements will require different installation methods (cargo,
    apt, etc.)
- uninstalling with instructions from UNINSTALL
- interface for:
    + uninstalling single packages
    + installing single packages
    + by default maybe install everything that is copied over(?), next run
      should omit already installed stuff based on the lockfile
- uninstalling should probably happen when an option is passed to uninstall every package that is ignored
  and previously wasn't ignored based on the lockfile
- option to copy over changes to configs that were done after deploying them with `copy` mode back to the configuration store



# `INSTALL` format

different ways to install packages:
- `system:<pkg>`, detects the right way to install package based on the running system
- `apt:<pkg>`, `pacman:<pkg>`, installs directly using one of those
- `bash:<instructions`, just executes the bash commands directly like a script
- specyfing many options for installation (if one fails or w/e) - TBD, for now maybe just parse the single option

# `UNINSTALL` format

maybe should be treated as an `additional` way to provide information on what
to remove, whatever can be guessed based on the installation method could be
supplemented by this, if not present then only what is guessed based on the
installation method is done

# `DEPENDENCIES` format

basically `INSTALL` format, but many lines for many dependencies (obv)
