# Fedora was deliberately chosen to have a more recent version of Wine than Debian.

# The last version of powershell that is working with wine is: https://github.com/PowerShell/PowerShell/releases/tag/v6.2.5
ARG POWERSHELL_VERSION="6.2.5"
ARG XMAKE_VERSION="3.0.6" # https://github.com/xmake-io/xmake/releases/download/v3.0.6/xmake-v3.0.6.win64.zip
ARG GIT_VERSION="2.52.0"
ARG SEVENZIP_VERSION="2501"

ARG WINEPREFIX=/opt/prefix

# =========================================================
# Stage 1: Populate wine prefix with tools
# =========================================================
FROM fedora:43 AS msvc-builder

ARG WINEPREFIX
ARG POWERSHELL_VERSION
ARG XMAKE_VERSION
ARG GIT_VERSION
ARG SEVENZIP_VERSION

RUN echo '#!/bin/bash' > /usr/bin/wine-add-path && \
    echo 'set -e' >> /usr/bin/wine-add-path && \
    echo 'NEW_ENTRY="$1"' >> /usr/bin/wine-add-path && \
    echo 'KEY="HKEY_CURRENT_USER\Environment"' >> /usr/bin/wine-add-path && \
    echo 'CURRENT=$(wine reg query "$KEY" /v PATH 2>/dev/null | tr -d "\r" | grep "REG_EXPAND_SZ" | sed "s/^.*REG_EXPAND_SZ\s*//")' >> /usr/bin/wine-add-path && \
    echo 'if [ -z "$CURRENT" ]; then' >> /usr/bin/wine-add-path && \
    echo '  FINAL="$NEW_ENTRY"' >> /usr/bin/wine-add-path && \
    echo 'else' >> /usr/bin/wine-add-path && \
    echo '  FINAL="$CURRENT;$NEW_ENTRY"' >> /usr/bin/wine-add-path && \
    echo 'fi' >> /usr/bin/wine-add-path && \
    echo 'wine reg add "$KEY" /v PATH /t REG_EXPAND_SZ /d "$FINAL" /f' >> /usr/bin/wine-add-path && \
    echo 'wineserver -w' >> /usr/bin/wine-add-path && \
    chmod +x /usr/bin/wine-add-path


# ---- System dependencies (build-time) ----
RUN dnf install -y \
    wine \
    wine-core \
    wine-common \
    msitools \
    python3 \
    cabextract \
    p7zip \
    p7zip-plugins \
    xz \
    git \
    ca-certificates \
    findutils \
    procps-ng \
    && dnf clean all

# ---- Wine configuration ----
ENV WINEPREFIX=${WINEPREFIX}
ENV WINEDEBUG=-all
ENV WINEDLLOVERRIDES=mscoree,mshtml=

RUN mkdir -p ${WINEPREFIX}

# Init wine
RUN wineboot --init && \
    wine reg add "HKCU\\Software\\Wine\\Version" /v "CurrentVersion" /d "10.0" /f && \
    wine reg add "HKCU\\Software\\Wine\\Version" /v "CurrentBuild" /d "19041" /f && \
    while pgrep wineserver > /dev/null; do sleep 1; done

WORKDIR /tmp
RUN git clone https://github.com/mstorsjo/msvc-wine
WORKDIR /tmp/msvc-wine

# Download + install MSVC inside the prefix
RUN mkdir -p "${WINEPREFIX}/drive_c/Program Files/MSVC"
RUN python3 vsdownload.py --accept-license --dest "${WINEPREFIX}/drive_c/Program Files/MSVC" && \
    ./install.sh "${WINEPREFIX}/drive_c/Program Files/MSVC"

# CONFIGURE MSVC ENVIRONMENT VARIABLES
RUN \
    MSVC_VER=$(ls "${WINEPREFIX}/drive_c/Program Files/MSVC/vc/tools/msvc/" | sort -V | tail -n1) && \
    SDK_VER=$(ls "${WINEPREFIX}/drive_c/Program Files/MSVC/kits/10/include/" | sort -V | tail -n1) && \
    if [ -z "$MSVC_VER" ]; then echo "ERROR: MSVC Not Found!"; exit 1; fi && \
    case "$MSVC_VER" in \
    14.1*) VS_VER="15.0" ;; \
    14.2*) VS_VER="16.0" ;; \
    14.3*|14.4*) VS_VER="17.0" ;; \
    *)     VS_VER="15.0"; echo "WARNING: Unknown MSVC version, defaulting to 15.0" ;; \
    esac && \
    VS_INSTALL_DIR="C:\Program Files\MSVC" && \
    MSVC_DIR="$VS_INSTALL_DIR\vc\tools\msvc\\$MSVC_VER" && \
    SDK_BASE="$VS_INSTALL_DIR\kits\10" && \
    SDK_INC="$SDK_BASE\include\\$SDK_VER" && \
    SDK_LIB="$SDK_BASE\lib\\$SDK_VER" && \
    CL_BIN_DIR="${WINEPREFIX}/drive_c/Program Files/MSVC/vc/tools/msvc/$MSVC_VER/bin/Hostx64/x64" && \
    if [ ! -f "$CL_BIN_DIR/cl.exe" ]; then echo "ERROR: cl.exe not found at $CL_BIN_DIR"; exit 1; fi && \
    FINAL_INCLUDE="$MSVC_DIR\atlmfc\include;$MSVC_DIR\include;$SDK_INC\shared;$SDK_INC\ucrt;$SDK_INC\um;$SDK_INC\winrt;$SDK_INC\km" && \
    FINAL_LIB="$MSVC_DIR\atlmfc\lib\x64;$MSVC_DIR\lib\x64;$SDK_LIB\ucrt\x64;$SDK_LIB\um\x64;$SDK_LIB\km\x64" && \
    BIN_PATH_MSVC="$MSVC_DIR\bin\Hostx64\x64" && \
    BIN_PATH_SDK="$SDK_BASE\bin\\$SDK_VER\x64" && \
    wine reg add "HKEY_CURRENT_USER\Environment" /v INCLUDE /t REG_EXPAND_SZ /d "$FINAL_INCLUDE" /f && \
    wine reg add "HKEY_CURRENT_USER\Environment" /v LIB /t REG_EXPAND_SZ /d "$FINAL_LIB" /f && \
    wine reg add "HKEY_CURRENT_USER\Environment" /v VSINSTALLDIR /t REG_SZ /d "$VS_INSTALL_DIR" /f && \
    wine reg add "HKEY_CURRENT_USER\Environment" /v VisualStudioVersion /t REG_SZ /d "$VS_VER" /f && \
    wine reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\VisualStudio\SxS\VS7" /v "$VS_VER" /t REG_SZ /d "$VS_INSTALL_DIR" /f && \
    CURRENT_PATH=$(wine reg query "HKEY_CURRENT_USER\Environment" /v PATH 2>/dev/null | tr -d "\r" | grep "REG_EXPAND_SZ" | sed "s/^.*REG_EXPAND_SZ\s*//") && \
    if [ -z "$CURRENT_PATH" ]; then NEW_PATH="$BIN_PATH_MSVC;$BIN_PATH_SDK"; else NEW_PATH="$CURRENT_PATH;$BIN_PATH_MSVC;$BIN_PATH_SDK"; fi && \
    wine reg add "HKEY_CURRENT_USER\Environment" /v PATH /t REG_EXPAND_SZ /d "$NEW_PATH" /f && \
    wine reg add "HKEY_CURRENT_USER\Software\Wine\DllOverrides" /v "vcruntime140" /t REG_SZ /d "native,builtin" /f && \
    wine reg add "HKEY_CURRENT_USER\Software\Wine\DllOverrides" /v "vcruntime140_1" /t REG_SZ /d "native,builtin" /f && \
    wineserver -w && \
    echo "Testing cl.exe..." && \
    wine cl.exe 2>&1 | grep "Microsoft"

# Install powershell
RUN curl -L -o /tmp/powershell.zip https://github.com/PowerShell/PowerShell/releases/download/v${POWERSHELL_VERSION}/PowerShell-${POWERSHELL_VERSION}-win-x64.zip
RUN mkdir -p "${WINEPREFIX}/drive_c/Program Files/Powershell/6"
RUN unzip /tmp/powershell.zip -d "${WINEPREFIX}/drive_c/Program Files/Powershell/6" || [ $? -eq 1 ]
RUN wine-add-path "C:\Program Files\Powershell\6"
RUN wineserver -w
RUN echo 'if (Get-Module -Name PSReadLine) { Remove-Module -Name PSReadLine }' > "${WINEPREFIX}/drive_c/Program Files/Powershell/6/profile.ps1"

# Install xmake
RUN curl -L -o /tmp/xmake.zip https://github.com/xmake-io/xmake/releases/download/v${XMAKE_VERSION}/xmake-v${XMAKE_VERSION}.win64.zip
RUN 7z x /tmp/xmake.zip -o/tmp/xmake
RUN cp -r /tmp/xmake/xmake/ "${WINEPREFIX}/drive_c/Program Files/Xmake"
RUN wine reg add 'HKEY_CURRENT_USER\Environment' /v XMAKE_PROGRAM_DIR /t REG_SZ /d 'C:\Program Files\Xmake' /f
RUN wine-add-path "C:\Program Files\Xmake"
RUN wineserver -w

# Install git
RUN curl -L -o /tmp/git.7z https://github.com/git-for-windows/git/releases/download/v${GIT_VERSION}.windows.1/PortableGit-${GIT_VERSION}-64-bit.7z.exe
RUN mkdir -p "${WINEPREFIX}/drive_c/Program Files/Git"
RUN 7z x /tmp/git.7z -o"${WINEPREFIX}/drive_c/Program Files/Git"
RUN wine-add-path "C:\Program Files\Git\bin"

# Install 7z
RUN curl -L -o /tmp/7z.7z https://www.7-zip.org/a/7z${SEVENZIP_VERSION}-extra.7z
RUN 7z x /tmp/7z.7z -o"${WINEPREFIX}/drive_c/Program Files/7-Zip"
RUN mv "${WINEPREFIX}/drive_c/Program Files/7-Zip/7za.exe" "${WINEPREFIX}/drive_c/Program Files/7-Zip/7z.exe"
RUN wine-add-path "C:\Program Files\7-Zip"

# Install rustup
# Pin to rustup 1.28.2 which is known to work in Wine
# RUN curl -L -o /tmp/rustup-init.exe https://win.rustup.rs/x86_64
RUN curl -L -o /tmp/rustup-init.exe https://static.rust-lang.org/rustup/archive/1.28.2/x86_64-pc-windows-msvc/rustup-init.exe
RUN wine Z:/tmp/rustup-init.exe -vy
RUN wine rustup set auto-self-update disable
RUN wineserver -w

# =========================================================
# Stage 2: Minimal RootFS Builder
# =========================================================
FROM fedora:43 AS rootfs-builder

# Create a directory to hold a minimal OS
RUN mkdir -p /mnt/rootfs

# Install strictly necessary dependencies into /mnt/rootfs
RUN dnf install -y \
    --installroot=/mnt/rootfs \
    --releasever=43 \
    --setopt=install_weak_deps=False \
    --setopt=tsflags=nodocs \
    --use-host-config \
    wine-core \
    wine-common \
    glibc-langpack-en \
    # Disable for now
    # bash \
    # coreutils \
    # grep \
    # sed \
    && dnf clean all --installroot=/mnt/rootfs

# Clean up dnf metadata/cache to save space
RUN rm -rf /mnt/rootfs/var/cache/dnf /mnt/rootfs/var/log/dnf*

# =========================================================
# Stage 3: Distroless Final Image
# =========================================================
FROM scratch

ARG WINEPREFIX

# Copy the minimal Fedora root filesystem
COPY --from=rootfs-builder /mnt/rootfs/ /

# Copy the populated Wine Prefix from the builder
COPY --from=msvc-builder ${WINEPREFIX} ${WINEPREFIX}

# Configure Environment
ENV WINEPREFIX=${WINEPREFIX}
ENV WINEDEBUG=-all
ENV XDG_RUNTIME_DIR=/tmp
ENV LANG=en_US.UTF-8

WORKDIR /work

# Default entrypoint
CMD ["wine", "cmd"]
