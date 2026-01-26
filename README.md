# Linux Subsystem for Windows

Recently, I started using Linux on my desktop, which made me realise something. WSL is extremely useful for building projects for Linux on Windows, but there is no such thing on Linux. That's why I made LSW the goal is to replicate WSL but in reverse.

Like WSL, it will have two versions: the first will be based on Wine, and the second will use Incus VM.

## Installation & Setup

### Install Dependencies

*Fedora:*
```bash
sudo dnf install incus wine
```

*Debian:*
LSW requires Incus >= 6.11. Since official Debian repositories often carry older versions, use the [Zabbly repository](github.com/zabbly/incus) for the latest builds.


Once Incus is installed, initialize it:
```bash
sudo incus admin init --auto
```

To avoid using `sudo` for every Incus command, add your user to the `incus-admin` group:
```bash
sudo usermod -aG incus-admin $USER
```

### ISO Utilities

LSW requires an ISO utility to package Windows components. Install *one* of the following:
* `mkisofs`
* `genisoimage`
* `xorriso`

## Getting started

You can create a new bottle (an instance of Windows tied to a v1 or v2 backend), by running 
```bash
lsw new v2
```

Offline Initialization:
By default, creating a bottle requires an internet connection (to download required components). If you want to prepare LSW for offline usage, you can initialize the required assets first:
```bash
lsw new --init
```
This command downloads all backend-specific dependencies *without* creating a bottle.

### Windows ISO Requirement

> [!IMPORTANT]  
> The Windows ISO cannot yet be downloaded automatically.  
> You must manually download a Windows ISO (currently tested with [Windows Server 23H2, no GUI](https://massgrave.dev/windows-server-links#windows-server-23h2-no-gui))  
> Place the ISO at one of the following locations:  
> * `~/.cache/lsw/downloads/windows-server.iso`  
> * `$XDG_CACHE_HOME/lsw/downloads/windows-server.iso`  
> 
> Make sure the filename is exactly:  
> ```
> windows-server.iso
> ```


# Development

For development, it’s best to use a devcontainer, for that I recommend using [devpod](https://github.com/loft-sh/devpod).

Before starting the container, ensure that virtualization is enabled and that the required kernel modules are loaded:
```bash
sudo modprobe vhost_vsock
sudo modprobe vhost_net
```
Then start the devcontainer:
```
devpod up --ide none .
```
You can connect to the container via SSH:
```
devpod ssh .
```

If you got an error because the incus network `incusbr0` is not created run `sudo incus admin init --auto`

## VM Debugging

A helper script named `connect-vm.sh` is provided to open a VM using virt-viewer.

You can modify this script with your specific VM name. This is particularly useful for debugging Windows installation and early boot issues.
