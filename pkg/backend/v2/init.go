package v2

import (
	"fmt"
	"os"
	"path"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/lxc/incus/shared/util"
	"github.com/plus3it/gorecurcopy"
)

func downloadOpenSSH() (string, error) {
	const version = "v8.1.0.0p1-Beta"
	url := fmt.Sprintf("https://github.com/PowerShell/Win32-OpenSSH/releases/download/%s/OpenSSH-Win64.zip", version)
	return backend.DownloadFileIfNeeded(url, "OpenSSH-Win64.zip")
}

func downloadWinFsp() (string, error) {
	const url = "https://github.com/winfsp/winfsp/releases/download/v2.1/winfsp-2.1.25156.msi"
	return backend.DownloadFileIfNeeded(url, "winfsp.msi")
}

func downloadVirtio() (string, error) {
	const url = "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/latest-virtio/virtio-win.iso"
	return backend.DownloadFileIfNeeded(url, "virtio.iso")
}

func downloadUnattendAssets() error {
	version := config.GetVersion()
	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/assets/v2", version.Commit)
	_, err := backend.DownloadFileIfNeeded(url+"/autounattend.xml", "autounattend.xml")
	if err != nil {
		return err
	}
	_, err = backend.DownloadFileIfNeeded(url+"/scripts/setup.ps1", "scripts/setup.ps1")
	if err != nil {
		return err
	}
	_, err = backend.DownloadFileIfNeeded(url+"/scripts/specialize.ps1", "scripts/specialize.ps1")
	return err
}

func downloadWindowsIso() (string, error) {
	// Taken from massgrave
	// https://massgrave.dev/windows-server-links#windows-server-23h2-no-gui
	// https://buzzheavier.com/e1ddpdjpxi0n/
	// Doesn't work might need to host it myself.
	return "", nil
	// return backend.DownloadFileIfNeeded("https://buzzheavier.com/e1ddpdjpxi0n/download", "windows-server.iso")
}

// Create a iso to install some software without internet connection
func createSoftwareISO(winfspPath string, openSSHPath string) error {
	cachedir, err := backend.GetCacheDir()
	if err != nil {
		log.Fatal("cannot get cache directory")
	}
	log.Debug(openSSHPath)

	tmpDir := path.Join(cachedir, "tmp", "software")
	isoPath := path.Join(cachedir, "iso", "software.iso")
	openSSHtmpDir := path.Join(tmpDir, "OpenSSH")

	if !util.PathExists(isoPath) {
		backend.CreateDir(tmpDir, 0755)
		backend.CreateDir(openSSHtmpDir, 0755)

		gorecurcopy.Copy(winfspPath, path.Join(tmpDir, "winfsp.msi"))
		gorecurcopy.CopyDirectory(openSSHPath, openSSHtmpDir)
		generateISO(tmpDir, isoPath, "SOFTWARE")

		os.RemoveAll(tmpDir)
	}

	return nil
}

// Download needed files
func Init() {
	openSSHPath, err := downloadOpenSSH()
	if err != nil {
		log.Fatal("cannot download OpenSSH")
	}

	winfspPath, err := downloadWinFsp()
	if err != nil {
		log.Fatal("cannot download WinFsp")
	}

	_, err = downloadVirtio()
	if err != nil {
		log.Fatal("cannot download Virtio")
	}

	err = downloadUnattendAssets()
	if err != nil {
		log.Fatal("cannot download Unattend assets")
	}

	_, err = downloadWindowsIso()
	if err != nil {
		log.Debug("cannot download Windows ISO, this is normal for now")
	}

	err = createSoftwareISO(winfspPath, openSSHPath)
	if err != nil {
		log.Fatal("cannot create software ISO")
	}
}
