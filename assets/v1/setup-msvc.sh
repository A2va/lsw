#!/bin/bash
set -e

echo "Configuring MSVC Environment..."
MSVC_VER=$(ls "${WINEPREFIX}/drive_c/Program Files/MSVC/vc/tools/msvc/" | sort -V | tail -n1)
SDK_VER=$(ls "${WINEPREFIX}/drive_c/Program Files/MSVC/kits/10/include/" | sort -V | tail -n1)

if [ -z "$MSVC_VER" ]; then echo "ERROR: MSVC Not Found!"; exit 1; fi

case "$MSVC_VER" in
  14.1*) VS_VER="15.0" ;;
  14.2*) VS_VER="16.0" ;;
  14.3*|14.4*) VS_VER="17.0" ;;
  14.5*|14.6*) VS_VER="18.0" ;;
  *)     VS_VER="18.0"; echo "WARNING: Unknown MSVC version, defaulting to 18.0" ;;
esac

# Create the suffix MSBuild expects (e.g., "18.0" -> "180")
VS_VER_SUFFIX="${VS_VER//./}"

VS_INSTALL_DIR="C:\Program Files\MSVC"
MSVC_DIR="$VS_INSTALL_DIR\vc\tools\msvc\\$MSVC_VER"
SDK_BASE="$VS_INSTALL_DIR\kits\10"
SDK_INC="$SDK_BASE\include\\$SDK_VER"
SDK_LIB="$SDK_BASE\lib\\$SDK_VER"

BIN_PATH_MSVC="$MSVC_DIR\bin\Hostx64\x64"
BIN_PATH_SDK="$SDK_BASE\bin\\$SDK_VER\x64"

FINAL_INCLUDE="$MSVC_DIR\atlmfc\include;$MSVC_DIR\include;$SDK_INC\shared;$SDK_INC\ucrt;$SDK_INC\um;$SDK_INC\winrt;$SDK_INC\km"
FINAL_LIB="$MSVC_DIR\atlmfc\lib\x64;$MSVC_DIR\lib\x64;$SDK_LIB\ucrt\x64;$SDK_LIB\um\x64;$SDK_LIB\km\x64"

# =================================================================
# CREATE vcvarsall.bat
# =================================================================
BAT_DIR="${WINEPREFIX}/drive_c/Program Files/MSVC/VC/Auxiliary/Build"
mkdir -p "$BAT_DIR"
BAT_FILE="$BAT_DIR/vcvarsall.bat"

cat <<EOF > "$BAT_FILE"
@echo off
set "VSINSTALLDIR=$VS_INSTALL_DIR\\"
set "VCINSTALLDIR=$VS_INSTALL_DIR\VC\\"
set "VCToolsVersion=$MSVC_VER"
set "WindowsSdkDir=$SDK_BASE\\"
set "WindowsSDKVersion=$SDK_VER"
set "WindowsSdkBinPath=$SDK_BASE\bin\\"
set "WindowsSdkVerBinPath=$SDK_BASE\bin\\$SDK_VER\\"
set "UCRTVersion=$SDK_VER"
set "UniversalCRTSdkDir=$SDK_BASE\\"
set "INCLUDE=$FINAL_INCLUDE"
set "LIB=$FINAL_LIB"
set "PATH=$BIN_PATH_MSVC;$BIN_PATH_SDK;%PATH%"
echo [vcvarsall.bat] Environment initialized for: 'x64'
EOF

sed -i 's/$/\r/' "$BAT_FILE"

# =================================================================
# CREATE vcvarsall.ps1
# =================================================================
PS1_FILE="$BAT_DIR/vcvarsall.ps1"

cat <<EOF > "$PS1_FILE"
\$env:VSINSTALLDIR = '${VS_INSTALL_DIR}\'
\$env:VCINSTALLDIR = '${VS_INSTALL_DIR}\VC\'
\$env:VCToolsVersion = '${MSVC_VER}'
\$env:WindowsSdkDir = '${SDK_BASE}\'
\$env:WindowsSDKVersion = '${SDK_VER}'
\$env:WindowsSdkBinPath = '${SDK_BASE}\bin\'
\$env:WindowsSdkVerBinPath = '${SDK_BASE}\bin\\${SDK_VER}\'
\$env:UCRTVersion = '${SDK_VER}'
\$env:UniversalCRTSdkDir = '${SDK_BASE}\'
\$env:INCLUDE = '${FINAL_INCLUDE}'
\$env:LIB = '${FINAL_LIB}'
\$env:PATH = 'C:\Program Files\Ninja;${BIN_PATH_MSVC};${BIN_PATH_SDK};' + \$env:PATH

\$env:DisableRegistryUse = 'true'
\$env:VCInstallDir_${VS_VER_SUFFIX} = '${VS_INSTALL_DIR}\VC\'
\$env:VCToolsInstallDir_${VS_VER_SUFFIX} = '${MSVC_DIR}\'
\$env:MicrosoftKitRoot = '${SDK_BASE}\'
\$env:SDKReferenceDirectoryRoot = '${SDK_BASE}\'
\$env:SDKExtensionDirectoryRoot = '${SDK_BASE}\'
\$env:MSBUILDSDKREFERENCEDIRECTORY = '${SDK_BASE}\'
\$env:MSBUILDMULTIPLATFORMSDKREFERENCEDIRECTORY = '${SDK_BASE}\'
\$env:WindowsSdkDir_10 = '${SDK_BASE}\'
\$env:UniversalCRTSdkDir_10 = '${SDK_BASE}\'
\$env:WindowsTargetPlatformVersion = '${SDK_VER}'
\$env:UCRTContentRoot = '${SDK_BASE}\'
\$env:NETFXKitsDir = '${SDK_BASE}\'
\$env:NETFXSDKDir = '${SDK_BASE}\'

Write-Host "[vcvarsall.ps1] Environment initialized for: 'x64'" -ForegroundColor Green
EOF

sed -i 's/$/\r/' "$PS1_FILE"


# =================================================================
# COMPILE vswhere.exe
# =================================================================
echo "Compiling mock vswhere.exe..."
mkdir -p "${WINEPREFIX}/drive_c/Program Files (x86)/Microsoft Visual Studio/Installer"
sed -i "s/__VS_VER__/$VS_VER/g" vswhere.c

WINEPATH="$BIN_PATH_MSVC;$BIN_PATH_SDK" \
INCLUDE="$FINAL_INCLUDE" \
LIB="$FINAL_LIB" \
wine cl.exe /nologo vswhere.c /Fe:"C:\Program Files (x86)\Microsoft Visual Studio\Installer\vswhere.exe"

# =================================================================
# DEPLOY C++ RUNTIME TO SYSTEM32 & SYSWOW64
# =================================================================

echo "Deploying MSVC Runtime DLLs to System32 (x64) and SysWOW64 (x86)..."

SYS32="${WINEPREFIX}/drive_c/windows/system32"
SYSWOW64="${WINEPREFIX}/drive_c/windows/syswow64"
MSVC_ROOT="${WINEPREFIX}/drive_c/Program Files/MSVC"

# Copy 64-bit (x64) DLLs to System32
find "$MSVC_ROOT" -type f -path "*/x64/*" ! -path "*/onecore/*" \( \
    -iname "msvcp140*.dll" -o \
    -iname "vcruntime140*.dll" -o \
    -iname "ucrtbase*.dll" -o \
    -iname "concrt140*.dll" -o \
    -iname "vccorlib140*.dll" \
\) -exec cp -f {} "$SYS32/" \; 2>/dev/null || true

# Copy 32-bit (x86) DLLs to SysWOW64
find "$MSVC_ROOT" -type f -path "*/x86/*" ! -path "*/onecore/*" \( \
    -iname "msvcp140*.dll" -o \
    -iname "vcruntime140*.dll" -o \
    -iname "ucrtbase*.dll" -o \
    -iname "concrt140*.dll" -o \
    -iname "vccorlib140*.dll" \
\) -exec cp -f {} "$SYSWOW64/" \; 2>/dev/null || true

# Register DLL Overrides so Wine uses the Microsoft versions instead of crashing
for dll in vcruntime140 vcruntime140_1 vcruntime140d vcruntime140_1d \
           msvcp140 msvcp140_1 msvcp140_2 msvcp140d msvcp140_1d msvcp140_2d \
           ucrtbase ucrtbased concrt140 concrt140d vccorlib140 vccorlib140d; do
    wine reg add "HKEY_CURRENT_USER\Software\Wine\DllOverrides" /v "$dll" /t REG_SZ /d "native,builtin" /f
done

# Add vcvarsall.bat to PATH
/usr/bin/wine-add-path "C:\Program Files\MSVC\VC\Auxiliary\Build"

# =================================================================
# Registry Edit
# =================================================================

wine reg add "HKEY_CURRENT_USER\Environment" /v VSINSTALLDIR /t REG_SZ /d "$VS_INSTALL_DIR" /f
wine reg add "HKEY_CURRENT_USER\Environment" /v VisualStudioVersion /t REG_SZ /d "$VS_VER" /f
wine reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\VisualStudio\SxS\VS7" /v "$VS_VER" /t REG_SZ /d "$VS_INSTALL_DIR" /f

# =================================================================
# POLYFILL ROBOCOPY.EXE
# =================================================================
# Some project uses robocopy but on wine it is just a empty stub, so implement a polyfill
# See:
# https://github.com/thewh1teagle/piper-rs/blob/616e6823cc29955bb0cf66532637b352b26fa292/crates/espeak-rs-sys/build.rs#L43
# https://gitlab.winehq.org/wine/wine/-/blob/master/programs/robocopy/main.c

echo "Compiling mock robocopy.exe..."

cat <<'EOF' > robocopy_mock.c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char** argv) {
    char* src = NULL;
    char* dst = NULL;

    // Parse arguments, ignoring flags like /e
    for(int i = 1; i < argc; i++) {
        if (argv[i][0] == '/') continue;
        if (!src) src = argv[i];
        else if (!dst) dst = argv[i];
    }

    // Redirect to xcopy
    if (src && dst) {
        char cmd[8192];
        // /E = Subdirectories (including empty)
        // /I = Assume destination is a directory
        // /Y = Suppress overwrite prompts
        // /Q = Quiet mode
        snprintf(cmd, sizeof(cmd), "xcopy \"%s\" \"%s\\\" /E /I /Y /Q", src, dst);
        return system(cmd);
    }
    return 0;
}
EOF

# Compile it into the System32 directory
WINEPATH="$BIN_PATH_MSVC;$BIN_PATH_SDK" \
INCLUDE="$FINAL_INCLUDE" \
LIB="$FINAL_LIB" \
wine cl.exe /nologo robocopy_mock.c /Fe:"${WINEPREFIX}/drive_c/windows/system32/robocopy.exe"

# Tell Wine to use our working version instead of its empty built-in stub
wine reg add "HKEY_CURRENT_USER\Software\Wine\DllOverrides" /v "robocopy.exe" /t REG_SZ /d "native,builtin" /f


wineserver -w
