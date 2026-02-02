function Get-DriveByFile {
    param([string]$FileName)
    # Check PSDrives first, then Volumes
    $drive = (Get-PSDrive | Where-Object { Test-Path "$($_.Name):\$FileName" }).Name
    if (-not $drive) {
        $drive = (Get-Volume | Where-Object { Test-Path "$($_.DriveLetter):\$FileName" }).DriveLetter
    }
    return $drive
}

function Install-VirtioTools {
    $drive = Get-DriveByFile "virtio-win-guest-tools.exe"
    if (-not $drive) {
        Write-Warning "VirtIO ISO not found."; return
    }

    Write-Host "Found VirtIO ISO on $drive. Trusting certificates..."
    $certFile = Get-ChildItem -Path "$($drive):\cert\*.cat" -Recurse | Select-Object -First 1
    if ($certFile) {
        certutil -addstore "TrustedPublisher" $certFile.FullName
    }

    Write-Host "Installing Guest Tools..."
    Start-Process -FilePath "$($drive):\virtio-win-guest-tools.exe" -ArgumentList "/passive", "/norestart" -Wait
    Start-Sleep -Seconds 10
}

# --- OPENSSH INSTALLATION ---

function Install-OpenSSH {
    $drive = Get-DriveByFile "OpenSSH\install-sshd.ps1"
    if (-not $drive) {
        Write-Warning "OpenSSH source not found on any drive."; return
    }

    $dest = "C:\OpenSSH"
    if (-not (Test-Path $dest)) { New-Item -ItemType Directory -Path $dest -Force }
    Copy-Item -Path "$($drive):\OpenSSH\*" -Destination $dest -Recurse -Force


    # Remove "Read-Only" attribute from all copied files
    Get-ChildItem -Path $dest -Recurse | ForEach-Object {
        if ($_.Attributes -match "ReadOnly") {
            $_.Attributes = 'Archive'
        }
    }

    # Run the official install script
    Set-Location $dest
    powershell.exe -ExecutionPolicy Bypass -File ".\install-sshd.ps1"

    $psPath = (Get-Command powershell.exe).Source
    New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value $psPath -PropertyType String -Force


    # Configure Service and Firewall
    Set-Service -Name sshd -StartupType 'Automatic'
    Start-Service sshd
    if (!(Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22
    }
}

function Install-Redistribuable {
    $drive = Get-DriveByFile "vc_redist.exe"
    if (-not $drive) {
        Write-Warning "vc_redist installer not found."; return
    }
    Start-Process -FilePath "$($drive):\vc_redist.exe" -ArgumentList "/install", "/passive", "/norestart"  -Wait
    Start-Sleep -Seconds 10
}

function Install-WinFSP {
    # Searches for a file matching winfsp-*.msi
    $drive = Get-DriveByFile "winfsp.msi"
    if (-not $drive) {
        Write-Warning "WinFSP installer not found."; return
    }

    $msiPath = Get-ChildItem -Path "$($drive):\winfsp.msi" | Select-Object -First 1
    Write-Host "Installing WinFSP from $($msiPath.FullName)..."

    Copy-Item $msiPath "C:\winfsp.msi"

    # INSTALLLEVEL=1000 ensures all features (including FUSE and Developer tools) are installed
    $arguments = "/i `"C:\winfsp.msi`" ADDLOCAL=ALL /qn /norestart"

    Start-Process "msiexec.exe" -ArgumentList $arguments -Wait
    Write-Host "WinFSP installation complete."
}

function Install-IncusAgent {
    $drive = Get-DriveByFile "incus-agent-setup.ps1"
    if (-not $drive) {
        Write-Warning "Incus installer not found."; return
    }
    powershell.exe -ExecutionPolicy Bypass -File "$($drive):\install.ps1"
    Write-Host "Incus Agent installation complete"
}

Install-IncusAgent
Install-Redistribuable
Install-WinFSP
Install-VirtioTools
Install-OpenSSH

Set-Service -Name "VirtioFsSvc" -StartupType Automatic
Start-Service -Name "VirtioFsSvc"

# Final Shutdown
Write-Output "Setup complete. Shutting down..."
Start-Sleep -Seconds 10
shutdown.exe /s /t 0 /f
