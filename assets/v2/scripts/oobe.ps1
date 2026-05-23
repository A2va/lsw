$LogFile = "C:\oobe.log"

function Write-Log {
    param (
        [Parameter(Mandatory=$true)]
        [string]$Message,

        [ValidateSet("INFO", "WARNING", "ERROR")]
        [string]$Level = "INFO"
    )
    $timestamp = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
    $logEntry = "[$timestamp] [$Level] $Message"

    # Append to log file
    Add-Content -Path $LogFile -Value $logEntry -ErrorAction SilentlyContinue

    # Output to console
    switch ($Level) {
        "INFO"    { Write-Host $logEntry }
        "WARNING" { Write-Host $logEntry -ForegroundColor Yellow }
        "ERROR"   { Write-Host $logEntry -ForegroundColor Red }
    }
}


function Get-DriveByFile {
    param([string]$FileName)
    try {
        $drive = Get-PSDrive -PSProvider FileSystem |
            Where-Object { Test-Path "$($_.Name):\$FileName" } |
            Select-Object -First 1 -ExpandProperty Name

        if (-not $drive) {
            $drive = Get-Volume |
                Where-Object { $_.DriveLetter -and (Test-Path "$($_.DriveLetter):\$FileName") } |
                Select-Object -First 1 -ExpandProperty DriveLetter
        }

        if ($drive) {
            Write-Log -Message "Found file '$FileName' on drive '$drive'."
        } else {
            Write-Log -Message "File '$FileName' not found on any drive." -Level WARNING
        }

        return $drive
    } catch {
        Write-Log -Message "Error locating drive for file '$FileName': $_" -Level ERROR
        return $null
    }
}

function Install-VirtioTools {
    try {
        Write-Log -Message "Starting VirtIO Guest Tools installation..."
        $drive = Get-DriveByFile "virtio-win-guest-tools.exe"
        if (-not $drive) {
            Write-Log -Message "VirtIO ISO not found. Skipping installation." -Level WARNING
            return
        }

        Write-Log -Message "Found VirtIO ISO on $drive. Trusting certificates..."
        $certFile = Get-ChildItem -Path "$($drive):\cert\*.cer" -Recurse | Select-Object -First 1
        if ($certFile) {
            Write-Log -Message "Found certificate: $($certFile.FullName)"

            $p = Start-Process certutil.exe `
                -ArgumentList @(
                    "-addstore",
                    "TrustedPublisher",
                    $certFile.FullName
                ) `
                -Wait -PassThru

            if ($p.ExitCode -eq 0) {
                Write-Log -Message "Certificate trusted: $($certFile.Name)"
            } else {
                Write-Log -Message "certutil failed with exit code $($p.ExitCode)" -Level ERROR
            }
        } else {
            Write-Log -Message "No certificate found to trust." -Level WARNING
        }

        Write-Log -Message "Executing VirtIO Guest Tools installer..."
        Start-Process -FilePath "$($drive):\virtio-win-guest-tools.exe" -ArgumentList "/passive", "/norestart" -Wait
        Start-Sleep -Seconds 10
        Write-Log -Message "VirtIO Guest Tools installation complete."
    } catch {
        Write-Log -Message "Error installing VirtIO Tools: $_" -Level ERROR
    }
}

function Install-OpenSSH {
    try {
        Write-Log -Message "Starting OpenSSH installation..."
        $drive = Get-DriveByFile "OpenSSH\install-sshd.ps1"
        if (-not $drive) {
            Write-Log -Message "OpenSSH source not found on any drive. Skipping installation." -Level WARNING
            return
        }

        $dest = "C:\OpenSSH"
        if (-not (Test-Path $dest)) {
            New-Item -ItemType Directory -Path $dest -Force | Out-Null
            Write-Log -Message "Created directory $dest."
        }

        Write-Log -Message "Copying OpenSSH files..."
        Copy-Item -Path "$($drive):\OpenSSH\*" -Destination $dest -Recurse -Force

        # Remove "Read-Only" attribute from all copied files
        Get-ChildItem -Path $dest -Recurse | ForEach-Object {
            if ($_.Attributes -match "ReadOnly") {
                $_.Attributes = 'Archive'
            }
        }
        Write-Log -Message "Removed Read-Only attributes from OpenSSH files."

        # Run the official install script
        Write-Log -Message "Running install-sshd.ps1..."
        Set-Location $dest
        Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy Bypass -File .\install-sshd.ps1" -Wait

        $psPath = (Get-Command powershell.exe).Source
        New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value $psPath -PropertyType String -Force | Out-Null
        Write-Log -Message "Set DefaultShell for OpenSSH to PowerShell."

        # Configure Service and Firewall
        Set-Service -Name sshd -StartupType 'Automatic'
        Start-Service sshd
        Write-Log -Message "Configured and started sshd service."

        if (!(Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue)) {
            New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 | Out-Null
            Write-Log -Message "Created firewall rule for OpenSSH."
        }

        Set-NetIPv6Protocol -RandomizeIdentifiers Disabled
        Set-NetIPv6Protocol -UseTemporaryAddresses Disabled
        Write-Log -Message "Disabled IPv6 randomized identifiers and temporary addresses."
        Write-Log -Message "OpenSSH installation complete."
    } catch {
        Write-Log -Message "Error installing OpenSSH: $_" -Level ERROR
    }
}

function Install-Redistribuable {
    try {
        Write-Log -Message "Starting VC Redistributable installation..."
        $drive = Get-DriveByFile "vc_redist.exe"
        if (-not $drive) {
            Write-Log -Message "vc_redist installer not found. Skipping." -Level WARNING
            return
        }

        Write-Log -Message "Executing vc_redist installer..."
        Start-Process -FilePath "$($drive):\vc_redist.exe" -ArgumentList "/install", "/passive", "/norestart" -Wait
        Start-Sleep -Seconds 10
        Write-Log -Message "VC Redistributable installation complete."
    } catch {
        Write-Log -Message "Error installing VC Redistributable: $_" -Level ERROR
    }
}

function Install-WinFSP {
    try {
        Write-Log -Message "Starting WinFSP installation..."
        # Searches for a file matching winfsp-*.msi
        $drive = Get-DriveByFile "winfsp.msi"
        if (-not $drive) {
            Write-Log -Message "WinFSP installer not found. Skipping." -Level WARNING
            return
        }

        $msiPath = "$drive`:\winfsp.msi"
        Write-Log -Message "Copying WinFSP from $msiPath to C:\..."
        Copy-Item $msiPath "C:\winfsp.msi" -Force

        # ADDLOCAL=ALL ensures all features (including FUSE and Developer tools) are installed
        $arguments = "/i `"C:\winfsp.msi`" ADDLOCAL=ALL /qn /norestart"

        Write-Log -Message "Executing WinFSP MSI installer..."
        Start-Process "msiexec.exe" -ArgumentList $arguments -Wait
        Write-Log -Message "WinFSP installation complete."
    } catch {
        Write-Log -Message "Error installing WinFSP: $_" -Level ERROR
    }
}

function Install-IncusAgent {
    try {
        Write-Log -Message "Starting Incus Agent installation..."
        $drive = Get-DriveByFile "incus-agent-setup.ps1"
        if (-not $drive) {
            Write-Log -Message "Incus installer script not found. Skipping." -Level WARNING
            return
        }

        # Fix because Fedora incus-agent package doesn't package windows agent binaries
        $incusAgent = Test-Path "$($drive):\incus-agent.exe"

        if (-not $incusAgent) {
            Write-Log -Message "incus-agent.exe not found on $drive, searching other drives..."
            New-Item "C:\Program Files\Incus-Agent" -ItemType Directory -Force | Out-Null

            $agentDrive = Get-DriveByFile "incus-agent.exe"
            if (-not $agentDrive) {
                Write-Log -Message "incus-agent.exe not found on any drive. Aborting Incus Agent install." -Level WARNING
                return
            }

            Write-Log -Message "Copying incus-agent.exe from $agentDrive to target directories..."
            Copy-Item "$($agentDrive):\incus-agent.exe" -Destination "C:\Program Files\Incus-Agent" -Force
            New-Item "C:\ProgramData\Incus-Agent" -ItemType Directory -Force | Out-Null
            Copy-Item "$($agentDrive):\incus-agent.exe" -Destination "C:\ProgramData\Incus-Agent" -Force
        }

        Write-Log -Message "Running install.ps1 for Incus Agent..."
        Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy Bypass -File `"$($drive):\install.ps1`"" -Wait
        Write-Log -Message "Incus Agent installation complete."
    } catch {
        Write-Log -Message "Error installing Incus Agent: $_" -Level ERROR
    }
}

try {
    # Initialize Log File
    if (-not (Test-Path $LogFile)) { New-Item -Path $LogFile -ItemType File -Force | Out-Null }
    Write-Log -Message "--- Starting OOBE Setup Script ---"

    Install-IncusAgent
    Install-Redistribuable
    Install-WinFSP
    Install-VirtioTools
    Install-OpenSSH

    # Configure Services
    Write-Log -Message "Configuring Incus-Agent service..."
    Set-Service -Name "Incus-Agent" -StartupType Automatic
    Start-Service -Name "Incus-Agent"
    Write-Log -Message "Incus-Agent service started."

    # Setup VirtIO FS via WinFSP
    $virtiofsPath = "C:\Program Files\Virtio-Win\VioFS\virtiofs.exe"
    $fsregPath = "C:\Program Files (x86)\WinFsp\bin\fsreg.bat"

    if (Test-Path $virtiofsPath) {
        Write-Log -Message "VirtIO FS executable found. Registering with WinFSP..."
        $arguments = @(
            "virtiofs"
            "`"$virtiofsPath`""
            "`"-t %1 -m %2`""
        )
        Start-Process -FilePath $fsregPath -ArgumentList $arguments -Wait
        Write-Log -Message "VirtIO FS registration completed."
    } else {
        Write-Log -Message "virtiofs.exe not found at $virtiofsPath. Skipping WinFSP registration." -Level WARNING
    }

    # Final Shutdown
    Write-Log -Message "Setup complete. Shutting down system in 10 seconds..."
    Start-Sleep -Seconds 10
    shutdown.exe /s /t 0 /f

} catch {
    Write-Log -Message "A critical error occurred in the main execution block: $_" -Level ERROR
}
