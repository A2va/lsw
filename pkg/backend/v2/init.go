package v2

import (
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	"github.com/charmbracelet/log"
)

func computeStateHash(files []string) string {
	// Sort to ensure deterministic hashing regardless of input order
	sort.Strings(files)

	h := sha256.New()
	for _, file := range files {
		io.WriteString(h, file)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

func downloadOpenSSH() error {
	const version = "v8.1.0.0p1-Beta"
	url := fmt.Sprintf("https://github.com/PowerShell/Win32-OpenSSH/releases/download/%s/OpenSSH-Win64.zip", version)
	return cache.Add("OpenSSH", url)
}

func downloadWinFsp() error {
	const url = "https://github.com/winfsp/winfsp/releases/download/v2.1/winfsp-2.1.25156.msi"
	return cache.Add("winfsp.msi", url)
}

func downloadVirtio() error {
	const url = "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/latest-virtio/virtio-win.iso"
	return cache.Add("virtio.iso", url)
}

func downloadIncusAgent() error {
	c, err := incusClient()
	if err != nil {
		return err
	}

	server, _, err := c.GetServer()
	if err != nil {
		return err
	}

	incusVersion := server.Environment.ServerVersion

	// Make sure that incus agent version match the system one
	url := fmt.Sprintf("https://github.com/lxc/incus/releases/download/v%s/bin.windows.incus-agent.x86_64.exe", incusVersion)
	return cache.Add("incus-agent.exe", url)
}

func downloadUnattendAssets() error {
	version := config.GetVersion()
	if version.Version == "dev" {
		return nil
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/A2va/lsw/%s/assets/v2", version.Commit)
	err := cache.Add(url+"/autounattend.xml", "autounattend.xml")
	if err != nil {
		return err
	}
	err = cache.Add(url+"/scripts/setup.ps1", "scripts/setup.ps1")
	if err != nil {
		return err
	}
	err = cache.Add(url+"/scripts/specialize.ps1", "scripts/specialize.ps1")
	return err
}

func downloadVsRedistribuable() error {
	const url = "https://aka.ms/vs/17/release/vc_redist.x64.exe"
	return cache.Add("vc_redist.exe", url)
}

func downloadWindowsIso() (string, error) {
	// Taken from massgrave
	// https://massgrave.dev/windows-server-links#windows-server-23h2-no-gui
	// https://buzzheavier.com/e1ddpdjpxi0n/
	// Doesn't work might need to host it myself.
	return "", nil
}

// Create a iso to install some software without internet connection
// filesInIso is an array of name file in the cache
func createSoftwareISO(filesInIso []string) error {

	var cachedFiles []string
	for _, file := range filesInIso {
		item, err := cache.Get(file)
		if err != nil {
			return err
		}
		cachedFiles = append(cachedFiles, item.Path)
	}

	cacheDir, err := cache.GetCacheDir()
	if err != nil {
		log.Fatal("cannot get cache directory")
	}

	stateHash := computeStateHash(cachedFiles)

	// Construct the URL that *would* be used if we generated it
	// We need this to calculate the expected file hash in the cache
	tmpFilename := fmt.Sprintf("software-%s.iso", stateHash)
	tmpIsoPath := filepath.Join(cacheDir, "tmp", tmpFilename)
	expectedUrl := "file://" + tmpIsoPath

	// Calculate the hash that AddFile adds to the filename
	// The file on disk will look like: software-<UrlHash>.iso
	expectedUrlHash := cache.Hash(expectedUrl)

	log.Debug("generated url and hash", "url", expectedUrl, "hash", expectedUrlHash)

	tmpDir := path.Join(cacheDir, "tmp", "software")

	targetName := "iso/software.iso"
	existingPath, err := cache.Get(targetName)

	shouldGenerate := false

	if err != nil {
		// File missing
		if cache.IsNotCached(err) {
			log.Debug("ddd")
			shouldGenerate = true
		} else {
			return err
		}
	}

	// File exists. But is it the RIGHT version?
	// We check if the existing filename contains our expected hash.
	if !strings.Contains(filepath.Base(existingPath.Path), expectedUrlHash) {
		shouldGenerate = true
	}

	if shouldGenerate {
		log.Debug("generate software iso")
		utils.CreateDir(tmpDir, 0755)
		// defer os.RemoveAll(tmpDir)

		err = cache.CopyFromCache(tmpDir, filesInIso)
		if err != nil {
			return err
		}

		err = generateISO(tmpDir, tmpIsoPath, "SOFTWARE")
		if err != nil {
			return err
		}

		return cache.Add("iso/software.iso", expectedUrl)
	}

	return nil
}

// Download needed files
func Init() {
	err := downloadOpenSSH()
	if err != nil {
		log.Fatal("cannot download OpenSSH")
	}

	err = downloadWinFsp()
	if err != nil {
		log.Fatal("cannot download WinFsp")
	}

	err = downloadVirtio()
	if err != nil {
		log.Fatal("cannot download Virtio")
	}

	err = downloadUnattendAssets()
	if err != nil {
		log.Fatal("cannot download Unattend assets")
	}

	_, err = downloadWindowsIso()
	if err != nil {
		log.Warn("cannot download Windows ISO (this is currently expected behavior)")
	}

	err = downloadVsRedistribuable()
	if err != nil {
		log.Fatal("cannot download Visual C++ Redistribuable")
	}

	err = downloadIncusAgent()
	if err != nil {
		log.Fatal("cannot download incus agent")
	}

	ss := []string{"winfsp.msi", "vc_redist.exe", "OpenSSH", "incus-agent.exe"}

	err = createSoftwareISO(ss)
	if err != nil {
		log.Fatal("cannot create software ISO: %w", err)
	}
}
