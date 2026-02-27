# Linux Subsystem for Windows

Recently, I started using Linux and one day hit a Windows-only CI failure with no Windows machine available. WSL lets you run Linux inside Windows — but there wasn’t a simple way to do the opposite. So I built LSW, it lets you spin up isolated Windows environments from Linux, each of these environments is called a bottle.

LSW provides two backends:
* v1 — Wine (container-based) → fast, lightweight
* v2 — Incus VM (full Windows VM) 

## Installation & Setup

### Install Dependencies for v1

Install either [docker](https://docs.docker.com/engine/install/) or [podman](https://podman.io/docs/installation#linux-distributions).

Make sure to enable the docker/podman socket:
```
systemctl --user enable --now podman.socket
sudo systemctl enable --now docker
```

### Install Dependencies for v2

*Fedora:*
```bash
sudo dnf install incus virtiofsd
```

*Debian:*
LSW requires Incus >= 6.11. Since official Debian repositories often carry older versions, use the [Zabbly repository](github.com/zabbly/incus) for the latest builds.

Once Incus is installed, initialize it:
```bash
sudo incus admin init --auto
sudo systemctl enable --now incus.service
```

To avoid using `sudo` for every Incus command, add your user to the `incus-admin` group:
```bash
sudo usermod -aG incus-admin $USER
```

If you want to have an internet connection, you might have to setup your [firewall](https://linuxcontainers.org/incus/docs/main/howto/network_bridge_firewalld/)
On system using firewalld (like Fedora) you can execute:
```bash
sudo firewall-cmd --zone=trusted --change-interface=incusbr0 --permanent
sudo firewall-cmd --reload
```

The v2 backend also requires an ISO utility to package Windows components. Install *one* of the following:
* `mkisofs`
* `genisoimage`
* `xorriso`

## Getting started

You can create a new bottle (an instance of Windows tied to a v1 or v2 backend), by running 
```bash
lsw new v2
lsw new v1 --name test # give a name to the bottle
```
> [!NOTE]
> The first creation of a bottle can take a while.
>
> LSW needs to download and prepare Windows components:
>
> - v1 (Wine): ~18 min the first time, ~2 min afterwards (cached)
> - v2 (VM): ~15 min each time
>
> This is normal and only happens during provisioning.

Once a bottle is created, you can get a shell into it with:
```bash
lsw shell test
```

Offline Initialization:
By default, creating a bottle requires an internet connection (to download required components). If you want to prepare LSW for offline usage, you can initialize the required assets first:
```bash
lsw new --init
```
This command downloads all backend-specific dependencies *without* creating a bottle.

### Windows ISO Requirement (only for v2 backend)

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

# FAQ

## What is the difference or limitations between the two backends ?

Before we continue, it is important to clarify that the v1 backend is container-based and supports both Docker and Podman. 
The v2, on the other hand, is a fully managed virtual machine provided by Incus. 

Generally speaking, v1 bottles are faster and use fewer resources, but installing and running software can be more challenging. This is also why we have decided to package some common development tools, such as MSVC, Xmake and rustup, in v1 bottles. For the v2 backend, however, you have to install these tools yourself.

Another limitation of the v2 backend is that you cannot mount multiple directories at the same time. Only the current working directory when executing 'lsw shell' will be mounted. There is also no support for global mounting (i.e. those in the config file).

## How to resize a v2 bottle ?

This is not possible to do it with lsw directly but you can use incus. First stop the bottle and execute:
```bash
incus config device set <bottle-name> root size=<new-size>
```
Then start the VM again and execute this inside it:
```powershell
$size = (Get-PartitionSupportedSize -DriveLetter C)
Resize-Partition -DriveLetter C -Size $size.SizeMax
```

## Where are bottle files stored?

- v1 bottles live inside your volumes (Docker/Podman)
- v2 bottles are managed by Incus

You can inspect them with:
```bash
incus list
docker volume ls
podman volume ls
```

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
