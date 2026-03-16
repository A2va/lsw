:: Inspired by https://schneegans.de/windows/unattend-generator/
@echo off
setlocal enabledelayedexpansion

echo --- STEP 1: Searching for Installation Media, XML, and Drivers ---
set "VIRTIO_DRIVE="
set "IMAGE_FILE="
set "XML_FILE="
set "OEM_FOLDER="
set "PEDRIVERS_FOLDER="

for %%d in (C D E F G H I J K L M N O P Q R S T U V W X Y Z) do (
    :: Look for Windows Image
    if exist "%%d:\sources\install.wim" set "IMAGE_FILE=%%d:\sources\install.wim"
    if exist "%%d:\sources\install.esd" set "IMAGE_FILE=%%d:\sources\install.esd"
    if exist "%%d:\sources\install.swm" (
        set "IMAGE_FILE=%%d:\sources\install.swm"
        set "SWM_PARAM=/SWMFile:%%d:\sources\install*.swm"
    )

    :: Look for Unattend XML and Extra Folders
    if exist "%%d:\autounattend.xml" set "XML_FILE=%%d:\autounattend.xml"
    if exist "%%d:\$OEM$" set "OEM_FOLDER=%%d:\$OEM$"
    if exist "%%d:\$WinPEDriver$" set "PEDRIVERS_FOLDER=%%d:\$WinPEDriver$"

    :: 3. Look for VirtIO Drivers
    if not defined VIRTIO_DRIVE (
        if exist "%%d:\vioscsi" (
            echo Found VirtIO media on %%d:
            set "VIRTIO_DRIVE=%%d"

            :: Load VirtIO drivers into WinPE RAM
            if exist "%%d:\vioscsi\w11\amd64\vioscsi.inf" drvload.exe "%%d:\vioscsi\w11\amd64\vioscsi.inf"
            if exist "%%d:\vioscsi\2k25\amd64\vioscsi.inf" drvload.exe "%%d:\vioscsi\2k25\amd64\vioscsi.inf"
            if exist "%%d:\viostor\w11\amd64\viostor.inf" drvload.exe "%%d:\viostor\w11\amd64\viostor.inf"
            if exist "%%d:\viostor\2k25\amd64\viostor.inf" drvload.exe "%%d:\viostor\2k25\amd64\viostor.inf"
        )
    )
)

:: Also try to find the XML file via the Setup Registry (if started by setup.exe)
for /f "tokens=3" %%t in ('reg.exe query HKLM\System\Setup /v UnattendFile 2^>nul') do ( if exist %%t set "XML_FILE=%%t" )

if not defined IMAGE_FILE echo Could not locate install.wim, install.esd or install.swm. & pause & exit /b 1

:: Load any standard WinPEDrivers into RAM
if defined PEDRIVERS_FOLDER (
    for /R "%PEDRIVERS_FOLDER%" %%f IN (*.inf) do drvload.exe "%%f"
)

echo --- STEP 2: Partitioning Disk 0 ---
>X:\diskpart.txt (
    echo SELECT DISK=0
    echo CLEAN
    echo CONVERT GPT
    echo CREATE PARTITION EFI SIZE=100
    echo FORMAT QUICK FS=FAT32 LABEL="System"
    echo ASSIGN LETTER=S
    echo CREATE PARTITION MSR SIZE=16
    echo CREATE PARTITION PRIMARY
    echo FORMAT QUICK FS=NTFS LABEL="Windows"
    echo ASSIGN LETTER=W
)
diskpart.exe /s X:\diskpart.txt || ( echo diskpart.exe encountered an error. & pause & exit /b 1 )

echo --- STEP 3: Applying Windows Image (Index 1) ---
dism.exe /Apply-Image /ImageFile:"%IMAGE_FILE%" %SWM_PARAM% /Index:1 /ApplyDir:W:\ /CheckIntegrity /Verify || ( echo dism.exe encountered an error. & pause & exit /b 1 )
bcdboot.exe W:\Windows /s S: || ( echo bcdboot.exe encountered an error. & pause & exit /b 1 )

:: Remove Recovery Partition to save space
if exist W:\Windows\System32\Recovery\winre.wim del W:\Windows\System32\Recovery\winre.wim

:: Copy Unattend.xml to the OS Drive so the Specialize Pass works!
if defined XML_FILE (
    mkdir W:\Windows\Panther
    copy "%XML_FILE%" W:\Windows\Panther\unattend.xml
)

:: Strip 8.3 file names
fsutil.exe 8dot3name set W: 1
fsutil.exe 8dot3name strip /s /f W:\

echo --- STEP 4: Injecting Drivers into Offline Windows ---
if defined VIRTIO_DRIVE (
    dism.exe /Add-Driver /Image:W:\ /Driver:"%VIRTIO_DRIVE%:\vioscsi" /Recurse
    dism.exe /Add-Driver /Image:W:\ /Driver:"%VIRTIO_DRIVE%:\viostor" /Recurse
)
if defined PEDRIVERS_FOLDER (
    dism.exe /Add-Driver /Image:W:\ /Driver:"%PEDRIVERS_FOLDER%" /Recurse
)

echo --- STEP 5: Copying OEM Folders ---
set "ROBOCOPY_ARGS=/E /XX /COPY:DAT /DCOPY:DAT /R:0"
if defined OEM_FOLDER (
    if exist "%OEM_FOLDER%\$$" robocopy.exe "%OEM_FOLDER%\$$" W:\Windows %ROBOCOPY_ARGS%
    if exist "%OEM_FOLDER%\$1" robocopy.exe "%OEM_FOLDER%\$1" W:\ %ROBOCOPY_ARGS%
    for %%d in (C D E F G H I J K L M N O P Q R S T U V W Y Z) do (
        if exist "%OEM_FOLDER%\%%d" robocopy.exe "%OEM_FOLDER%\%%d" %%d:\ %ROBOCOPY_ARGS%
    )
)

echo --- Installation Complete. Rebooting ---
wpeutil.exe reboot
