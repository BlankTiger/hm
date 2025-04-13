Done:
- basic copying
- symlinking of directories (dev mode) for when I don't want to run `hm` everytime I'm incrementally
  working on some config
- hidden directories are not managed by `hm`
- ignore `.git`

Bugs:
- in case of systemd, if I want to store the enabled targets, then they are always stored as
  symlinks, currently this obviously only works in non `dev` mode, because of that maybe it's smart
  not to store systemd config at first (removing the directory when copying would also reset enabled
  targets because of this)

TODO:
- installing with instructions from INSTALL
    + installing dependencies from REQUIREMENTS files
    + parsing REQUIREMENTS -> some requirements will require different installation methods (cargo,
    apt, etc.)
- uninstalling with instructions from UNINSTALL
- lockfile, need to probably implement some kind of hashing to not overwrite files if there are some
  new ones in directories that are managed with `hm` (this shouldn't happen though, because everything
  should be managed via `hm`)
    + persist information on what was copied, what's managed
    + info on what was installed
