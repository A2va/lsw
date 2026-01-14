#!/bin/bash

VM_NAME="win"
SPICE_PORT="6000"
SOCKET_PATH="/run/incus/$VM_NAME/qemu.spice"

echo ">>> Configuring VM and Display inside DevContainer..."

# The commands needs to be ran on the container
devpod ssh . --command "
    # Wait for the SPICE socket to appear (Timeout after 10s)
    echo 'Waiting for VM display socket...'
    timeout 10s bash -c 'until sudo test -S $SOCKET_PATH; do sleep 0.5; done'

    # Check again with sudo
    if ! sudo test -S $SOCKET_PATH; then
        echo 'Error: Socket did not appear (or permission denied).'
        echo 'Debug: checking permissions...'
        sudo ls -la /run/incus/$VM_NAME/
        exit 1
    fi


    # Start/Restart the forwarding service
    if ! systemctl is-active --quiet win11-display; then
        echo 'Starting new display proxy...'
        sudo systemctl reset-failed win11-display 2>/dev/null || true

        # We use 'systemd-run' to keep it alive in the background
        sudo systemd-run --unit=win11-display \
            --description='Forward Win11 SPICE' \
            --service-type=simple \
            --property=Restart=always \
            --property=RestartSec=1 \
            socat TCP-LISTEN:$SPICE_PORT,fork,reuseaddr UNIX-CONNECT:$SOCKET_PATH
    else
        echo 'Display proxy is already running.'
    fi
"

echo ">>> Launching Viewer..."

if command -v remote-viewer &> /dev/null; then
    remote-viewer spice://localhost:$SPICE_PORT
else
    echo "ERROR: 'remote-viewer' not found."
    echo "Please connect manually to: spice://localhost:$SPICE_PORT"
fi
