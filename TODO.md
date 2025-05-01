Bugs:
- in case of systemd, if I want to store the enabled targets, then they are always stored as
  symlinks, currently this obviously only works in non `dev` mode, because of that maybe it's smart
  not to store systemd config at first (removing the directory when copying would also reset enabled
  targets because of this)

TODO:

- probably a good idea to parse all lines in INSTALL and execute them one by one until one succeeds
has to depend on the lockfile (we have to store information on how the package was installed)
- option to copy over changes to configs that were done after deploying them with `copy` mode back to the configuration store
- would be nice to calculate installation dependency graph such that everything always gets installed in the correct order
