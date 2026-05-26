$LogFile = "C:\oobe.log"

# ---------------------------------------------------------
# Helper Functions
# ---------------------------------------------------------

function Write-Log {
    param (
        [Parameter(Mandatory=$true)]
        [string]$Message,

        [ValidateSet("INFO", "WARNING", "ERROR")]
        [string]$Level = "INFO"
    )
    $timestamp = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
    $logEntry = "[$timestamp] [$Level] $Message"

    Add-Content -Path $LogFile -Value $logEntry -ErrorAction SilentlyContinue

    switch ($Level) {
        "INFO"    { Write-Host $logEntry }
        "WARNING" { Write-Host $logEntry -ForegroundColor Yellow }
        "ERROR"   { Write-Host $logEntry -ForegroundColor Red }
    }
}

function Invoke-CommandWithLogging {
    param (
        [Parameter(Mandatory=$true)][string]$FilePath,
        [string[]]$ArgumentList = @(),
        [Parameter(Mandatory=$true)][string]$Name,
        [string]$WorkingDirectory = "",
        [int[]]$ValidExitCodes = @(0, 3010, 1638),
        [string]$LogPath = ""
    )

    Write-Log -Message "Executing: $Name"

    $outFile = [System.IO.Path]::GetTempFileName()
    $errFile = [System.IO.Path]::GetTempFileName()

    $processArgs = @{
        FilePath               = $FilePath
        ArgumentList           = $ArgumentList
        Wait                   = $true
        PassThru               = $true
        NoNewWindow            = $true
        RedirectStandardOutput = $outFile
        RedirectStandardError  = $errFile
    }
    if ($WorkingDirectory) { $processArgs.WorkingDirectory = $WorkingDirectory }

    try {
        $process = Start-Process @processArgs

        if ($process.ExitCode -notin $ValidExitCodes) {
            Write-Log -Message "$Name failed with exit code $($process.ExitCode)." -Level ERROR

            # Capture and Log Standard Output / Standard Error
            $stdout = Get-Content $outFile -Raw -ErrorAction SilentlyContinue
            $stderr = Get-Content $errFile -Raw -ErrorAction SilentlyContinue

            if (-not [string]::IsNullOrWhiteSpace($stdout)) { Write-Log -Message "$Name STDOUT:`n$($stdout.Trim())" -Level ERROR }
            if (-not [string]::IsNullOrWhiteSpace($stderr)) { Write-Log -Message "$Name STDERR:`n$($stderr.Trim())" -Level ERROR }

            # For msiexec, stdout/stderr is usually empty. We must read the custom MSI log.
            if ($LogPath -and (Test-Path $LogPath)) {
                $tail = Get-Content $LogPath -Tail 30 -ErrorAction SilentlyContinue | Out-String
                Write-Log -Message "$Name Log Tail:`n$tail" -Level ERROR
            }
        } else {
            Write-Log -Message "$Name completed successfully."
        }
    } catch {
        Write-Log -Message "Failed to start process '$Name': $_" -Level ERROR
    } finally {
        if (Test-Path $outFile) { Remove-Item $outFile -Force -ErrorAction SilentlyContinue }
        if (Test-Path $errFile) { Remove-Item $errFile -Force -ErrorAction SilentlyContinue }
    }
}

function Start-ConfiguredService {
    param([Parameter(Mandatory=$true)][string]$ServiceName)

    Write-Log -Message "Configuring and starting service: $ServiceName"
    try {
        Set-Service -Name $ServiceName -StartupType Automatic -ErrorAction Stop
        Start-Service -Name $ServiceName -ErrorAction Stop
        Write-Log -Message "Service '$ServiceName' started successfully."
    } catch {
        # $_.Exception.Message extracts the exact reason Windows rejected the service start
        Write-Log -Message "Service '$ServiceName' failed: $($_.Exception.Message)" -Level ERROR
    }
}

function Get-DriveByFile {
    param([string]$FileName)
    try {
        $drive = Get-PSDrive -PSProvider FileSystem | Where-Object { Test-Path "$($_.Name):\$FileName" } | Select-Object -First 1 -ExpandProperty Name
        if (-not $drive) {
            $drive = Get-Volume | Where-Object { $_.DriveLetter -and (Test-Path "$($_.DriveLetter):\$FileName") } | Select-Object -First 1 -ExpandProperty DriveLetter
        }
        if ($drive) { Write-Log -Message "Found file '$FileName' on drive '$drive'." }
        else { Write-Log -Message "File '$FileName' not found on any drive." -Level WARNING }
        return $drive
    } catch {
        Write-Log -Message "Error locating drive for file '$FileName': $_" -Level ERROR
        return $null
    }
}

# ---------------------------------------------------------
# Installation Functions
# ---------------------------------------------------------

function Install-VirtioTools {
    $drive = Get-DriveByFile "virtio-win-guest-tools.exe"
    if (-not $drive) { return }

    $certFile = Get-ChildItem -Path "$($drive):\cert\*.cer" -Recurse | Select-Object -First 1
    if ($certFile) {
        Invoke-CommandWithLogging -FilePath "certutil.exe" -ArgumentList "-addstore", "TrustedPublisher", "`"$($certFile.FullName)`"" -Name "VirtIO Certificate Trust" -LogPath "C:\virtio_cert.log"
    }

    Invoke-CommandWithLogging -FilePath "$($drive):\virtio-win-guest-tools.exe" -ArgumentList "/passive", "/norestart" -Name "VirtIO Guest Tools" -LogPath "C:\virtio.log"
    $viosockInf = Get-ChildItem -Path "$($drive):\viosock\*\amd64\viosock.inf" | Sort-Object FullName -Descending | Select-Object -First 1
    if ($viosockInf) {
        Write-Log -Message "Found viosock driver at $($viosockInf.FullName). Installing..."
        Invoke-CommandWithLogging -FilePath "pnputil.exe" -ArgumentList "/add-driver", "`"$($viosockInf.FullName)`"", "/install" -Name "VirtIO Vsock Driver" -LogPath "C:\viosock.log"
    } else {
        Write-Log -Message "Could not find viosock.inf on VirtIO drive." -Level WARNING
    }

    Start-Sleep -Seconds 10
}

function Install-OpenSSH {
    $drive = Get-DriveByFile "OpenSSH\install-sshd.ps1"
    if (-not $drive) { return }

    $dest = "C:\OpenSSH"
    if (-not (Test-Path $dest)) { New-Item -ItemType Directory -Path $dest -Force | Out-Null }

    Copy-Item -Path "$($drive):\OpenSSH\*" -Destination $dest -Recurse -Force
    Get-ChildItem -Path $dest -Recurse | Where-Object Attributes -match "ReadOnly" | ForEach-Object { $_.Attributes = 'Archive' }

    Invoke-CommandWithLogging -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy", "Bypass", "-File", ".\install-sshd.ps1" -Name "OpenSSH Installer Script" -WorkingDirectory $dest -LogPath "C:\openssh.log"

    $psPath = (Get-Command powershell.exe).Source
    New-ItemProperty -Path "HKLM:\SOFTWARE\OpenSSH" -Name DefaultShell -Value $psPath -PropertyType String -Force | Out-Null

    Start-ConfiguredService -ServiceName "sshd"

    if (!(Get-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 | Out-Null
        Write-Log -Message "Created firewall rule for OpenSSH."
    }

    Set-NetIPv6Protocol -RandomizeIdentifiers Disabled -UseTemporaryAddresses Disabled
    Write-Log -Message "Disabled IPv6 randomized identifiers and temporary addresses."
}

function Install-Redistribuable {
    $drive = Get-DriveByFile "vc_redist.exe"
    if (-not $drive) { return }

    Invoke-CommandWithLogging -FilePath "$($drive):\vc_redist.exe" -ArgumentList "/install", "/passive", "/norestart" -Name "VC Redistributable" -LogPath "C:\vcredist.log"
    Start-Sleep -Seconds 10
}

function Install-WinFSP {
    $drive = Get-DriveByFile "winfsp.msi"
    if (-not $drive) { return }

    $msiPath = "C:\winfsp.msi"
    $msiLog = "C:\winfsp_install.log"

    Copy-Item "$drive`:\winfsp.msi" $msiPath -Force

    $args = @("/i", "`"$msiPath`"", "ADDLOCAL=ALL", "/qn", "/norestart", "/l*v", "`"$msiLog`"")
    Invoke-CommandWithLogging -FilePath "msiexec.exe" -ArgumentList $args -Name "WinFSP MSI" -LogPath $msiLog
}

function Install-IncusAgent {
    $drive = Get-DriveByFile "incus-agent-setup.ps1"
    if (-not $drive) { return }

    if (-not (Test-Path "$($drive):\incus-agent.exe")) {
        $agentDrive = Get-DriveByFile "incus-agent.exe"
        if (-not $agentDrive) { return }

        New-Item "C:\Program Files\Incus-Agent", "C:\ProgramData\Incus-Agent" -ItemType Directory -Force | Out-Null
        Copy-Item "$($agentDrive):\incus-agent.exe" -Destination "C:\Program Files\Incus-Agent" -Force
        Copy-Item "$($agentDrive):\incus-agent.exe" -Destination "C:\ProgramData\Incus-Agent" -Force
    }

    $args = @("-ExecutionPolicy", "Bypass", "-File", "`"$($drive):\install.ps1`"")
    Invoke-CommandWithLogging -FilePath "powershell.exe" -ArgumentList $args -Name "Incus Agent Setup Script" -LogPath "C:\incus-agent.log"
}

function Install-Chocolatey {
    $drive = Get-DriveByFile "chocolatey.msi"
    if (-not $drive) { return }

    $msiLog = "C:\chocolatey_install.log"
    $args = @("/i", "`"$drive`:\chocolatey.msi`"", "/qn", "/norestart", "/l*v", "`"$msiLog`"")

    Invoke-CommandWithLogging -FilePath "msiexec.exe" -ArgumentList $args -Name "Chocolatey MSI" -LogPath $msiLog
}

# ---------------------------------------------------------
# Main Execution Block
# ---------------------------------------------------------

try {
    if (-not (Test-Path $LogFile)) { New-Item -Path $LogFile -ItemType File -Force | Out-Null }
    Write-Log -Message "--- Starting OOBE Setup Script ---"

    Install-Redistribuable
    Install-WinFSP
    Install-VirtioTools
    Install-IncusAgent
    Install-OpenSSH
    Install-Chocolatey

    Start-ConfiguredService -ServiceName "Incus-Agent"

    # Setup VirtIO FS via WinFSP
    $virtiofsPath = "C:\Program Files\Virtio-Win\VioFS\virtiofs.exe"
    $fsregPath = "C:\Program Files (x86)\WinFsp\bin\fsreg.bat"

    if (Test-Path $virtiofsPath) {
        $args = @("virtiofs", "`"$virtiofsPath`"", "`"-t %1 -m %2`"")
        Invoke-CommandWithLogging -FilePath $fsregPath -ArgumentList $args -Name "VirtIO FS WinFSP Registration" -LogPath "C:\register_virtiofs.log"
    } else {
        Write-Log -Message "virtiofs.exe not found at $virtiofsPath. Skipping WinFSP registration." -Level WARNING
    }

    Write-Log -Message "Setup complete. Shutting down system in 10 seconds..."
    Start-Sleep -Seconds 10
    shutdown.exe /s /t 0 /f

} catch {
    Write-Log -Message "A critical error occurred in the main execution block: $_" -Level ERROR
}
