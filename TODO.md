Done:
- basic copying

TODO:
- symlinking of directories (dev mode) for when I don't want to run `hm` everytime I'm incrementally
  working on some config
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
