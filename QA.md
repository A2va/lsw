# Testing

This provides a regression testing before doing a release, this is also the checks that needs to be implemented in the CI.

## v1 (Wine/Container-based)

- [ ] `lsw new v1` creates a new v1 bottle successfully
- [ ] `lsw ps --all` shows the newly created bottle
- [ ] `lsw shell <name>` provides access to Windows shell
- [ ] Create a file within the Windows shell, verify it exists on the Linux host
- [ ] `lsw mount <name> <host-path>` mounts a host directory into the bottle
- [ ] After mounting, files are accessible from within the bottle
- [ ] `lsw remove <name>` removes the bottle and its resources

## v2 (Incus VM)

- [ ] `lsw new v2` creates a new v2 bottle successfully
- [ ] `lsw ps` shows the newly created bottle
- [ ] `lsw shell <name>` provides access to Windows shell
- [ ] Create a file within the Windows shell, verify it exists on the Linux host (via incus)
- [ ] `lsw stop <name>` stops the bottle gracefully
- [ ] `lsw start <name>` starts the stopped bottle
- [ ] `lsw mount <name> <host-path>` mounts a host directory into the bottle
- [ ] After mounting, files are accessible from within the bottle
- [ ] `lsw remove <name>` removes the bottle and its resources

## General

- [ ] `lsw ps --all` shows all bottles (running and stopped)
- [ ] Default bottle configuration works correctly
- [ ] Offline initialization `lsw new --init` works without network
