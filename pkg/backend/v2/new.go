package v2

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"

	"github.com/plus3it/gorecurcopy"
)

type NewArgument struct {
	Name     string
	Ram      string
	Disk     string
	Cpus     string
	Password string
	Username string
}

type WindowsLocales struct {
	InputLocale  string // Keyboard Layout (e.g., fr-CH)
	SystemLocale string // The OS Language (e.g., en-US)
	UserLocale   string // Date/Currency formats (e.g., fr-CH)
	UILanguage   string // The Display Language (e.g., en-US)
}

// mapLinuxLayoutToWindows converts X11 layout/variant to Windows Language-Region
func mapLinuxLayoutToWindows(layout, variant string) string {
	if layout == "" {
		return "en-US"
	}

	if variant == "" {
		return "en-US"
	}

	if layout == "ch" {
		if variant == "fr" {
			return "fr-CH"
		}
		return "de-CH"
	}

	if layout == "ca" {
		if variant == "fr" {
			return "fr-CA"
		}
		return "en-CA"
	}

	commonMap := map[string]string{
		"us": "en-US",
		"gb": "en-GB",
		"uk": "en-GB",
		"fr": "fr-FR",
		"de": "de-DE",
		"it": "it-IT",
		"es": "es-ES",
		"pt": "pt-PT",
		"jp": "ja-JP",
		"ru": "ru-RU",
	}

	if val, ok := commonMap[layout]; ok {
		return val
	}

	// Fallback
	return "en-US"
}

// Extract value from localectl output key
func getValue(data, key string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// Handle "LANG=en_US.UTF-8" -> "en-US"
func cleanLocaleEnv(env string) string {
	if env == "LANG=C.UTF-8" {
		return "en-US"
	}

	// Remove "LANG=" if present
	env, _ = strings.CutPrefix(env, "LANG=")

	// Split by dot (remove encoding)
	parts := strings.Split(env, ".")
	base := parts[0]

	// Convert underscore to dash (en_US -> en-US)
	return strings.ReplaceAll(base, "_", "-")
}

// Get primary keyboard on the system
func getPrimary(val string) string {
	parts := strings.Split(val, ",")
	return strings.TrimSpace(parts[0])
}

func detectLocale() (WindowsLocales, error) {
	out, err := exec.Command("localectl").Output()
	if err != nil {
		return WindowsLocales{}, err
	}

	data := string(out)

	// Parse Raw Data from localectl
	lang := getValue(data, "System Locale")     // e.g., LANG=en_US.UTF-8
	x11Layout := getValue(data, "X11 Layout")   // e.g., ch,ch
	x11Variant := getValue(data, "X11 Variant") // e.g., fr,fr_nodeadkeys

	// Clean up parsed data (remove LANG=, remove .UTF-8, handle commas)
	sysLang := cleanLocaleEnv(lang)
	primaryLayout := getPrimary(x11Layout)
	primaryVariant := getPrimary(x11Variant)

	// Determine InputLocale (Keyboard)
	// This requires mapping Linux layout+variant to Windows Culture codes
	inputLocale := mapLinuxLayoutToWindows(primaryLayout, primaryVariant)

	// Often it is safer to use the InputLocale region for UserLocale
	// so dates match the keyboard expectations, or just use SystemLocale.
	return WindowsLocales{
		InputLocale:  inputLocale,
		SystemLocale: sysLang,
		UILanguage:   sysLang,
		UserLocale:   inputLocale,
	}, nil
}

func getUnattendXmlFile() (string, error) {
	version := config.GetVersion()
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		return path.Join(wd, "assets", "v1", "autounattend.xml"), nil
	}
	cache, err := backend.GetCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(cache, "downloads", "autounattend.xml"), nil
}

// Copy the assets for creating a autounattend iso file to a temp dir
func copyUnattendAssetsToDir(d string) error {
	copyToDirFromCache := func(cachedir string, dir string, file string) {
		err := backend.CreateDir(path.Join(dir, path.Dir(file)), 0755)
		if err != nil {
			log.Fatal("error creating directory", "err", err)
		}

		dir = path.Join(dir, file)
		cachedir = path.Join(cachedir, file)
		err = gorecurcopy.Copy(cachedir, dir)
		if err != nil {
			log.Fatal("error copying file", "err", err)
		}
	}

	log.Debug("temp directory", "dir", d)

	cache, err := backend.GetCacheDir()
	if err != nil {
		return err
	}
	downloadCache := path.Join(cache, "downloads")

	version := config.GetVersion()
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		wd = path.Join("assets", "v2")
		// copyToDirFromCache(wd, d, "autounattend.xml")
		copyToDirFromCache(wd, d, "scripts/setup.ps1")
		copyToDirFromCache(wd, d, "scripts/specialize.ps1")
	} else {
		// copyToDirFromCache(downloadCache, d, "autounattend.xml")
		copyToDirFromCache(downloadCache, d, "scripts/setup.ps1")
		copyToDirFromCache(downloadCache, d, "scripts/specialize.ps1")
	}

	return nil
}

func createAutounattendISO(args NewArgument) (string, error) {
	tmpDir, err := os.MkdirTemp("", "lsw-autounattend")
	if err != nil {
		return "", err
	}

	isoPath := path.Join(tmpDir, "autounattend.iso")

	err = copyUnattendAssetsToDir(tmpDir)
	if err != nil {
		log.Fatal("error when copying assets to temporary directory", "err", err)
	}

	locale, err := detectLocale()
	if err != nil {
		log.Fatal("error when calling localectl", "err", err)
	}

	log.Debug("windows layout", "locale", locale)

	config := map[string]string{
		"Password":     args.Password,
		"Username":     args.Username,
		"UILanguage":   locale.UILanguage,
		"InputLocale":  locale.InputLocale,
		"SystemLocale": locale.SystemLocale,
		"UserLocale":   locale.UserLocale,
	}

	unattendXml, err := getUnattendXmlFile()
	if err != nil {
		return "", err
	}

	tmpl, _ := template.ParseFiles(unattendXml)
	f, _ := os.Create(path.Join(tmpDir, "autounattend.xml"))
	defer f.Close()

	tmpl.Execute(f, config)

	err = generateISO(tmpDir, isoPath, "AUTOUNATTEND")

	if err != nil {
		return "", err
	}

	return isoPath, nil
}

// sendMonitorKeys dials the QEMU monitor and sends the key command
func sendMonitorKeys(key string, monitorAddr string, count int) {
	log.Debug("action: sending key to monitor", "key", key, "count", count)
	conn, err := net.DialTimeout("tcp", monitorAddr, 5*time.Second)
	if err != nil {
		log.Warn("could not connect to QEMU monitor", "err", err)
		return
	}
	defer conn.Close()

	for range count {
		_, err := fmt.Fprintf(conn, "sendkey %s\n", key)
		if err != nil {
			log.Debug("error during sendkey", "err", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func New(arch string, args NewArgument) error {
	Init()

	log.Debug("new bottle on v1 backend", "name", args.Name)

	if arch != "amd64" {
		log.Fatal("not supported architecture")
	}

	// Create the autounattend iso
	isoPath, err := createAutounattendISO(args)
	if err != nil {
		return err
	}
	// defer os.RemoveAll(path.Dir(isoPath))

	c, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to incus socket: %w", err)
	}

	cache, err := backend.GetCacheDir()
	if err != nil {
		log.Fatal("cannot get cache dir")
	}
	downloadsDir := path.Join(cache, "downloads")
	isoDir := path.Join(cache, "iso")

	// FIXME might need to check if the port is available
	const monitorAddr = "127.0.0.1:12345"

	instance := api.InstancesPost{
		Name: args.Name,
		Type: api.InstanceTypeVM,
		Source: api.InstanceSource{
			Type: "none",
		},
		InstancePut: api.InstancePut{
			Config: map[string]string{
				"limits.cpu":    "4",
				"limits.memory": "4GiB",
				// Expose QEMU monitor via TCP to send the "any key" bypass
				"raw.qemu":      "-device intel-hda -device hda-duplex -audio spice -monitor tcp:" + monitorAddr + ",server,nowait",
				"raw.qemu.conf": "[audiodev \"qemu_spice-audiodev\"]\ndriver = \"none\"",
			},
			Devices: map[string]map[string]string{
				"root": {
					"type": "disk",
					"pool": "default",
					"path": "/",
					"size": "25GiB",
				},
				"vtpm": {"type": "tpm"},
				"eth0": {
					"type":    "nic",
					"network": "incusbr0",
				},
				"install": {
					"type":          "disk",
					"source":        path.Join(downloadsDir, "windows-server.iso"),
					"boot.priority": "10",
				},
				// ISO containing SSH and winfsp
				"software": {
					"type":   "disk",
					"io.bus": "usb",
					"source": path.Join(isoDir, "software.iso"),
				},
				"autounattend": {
					"type":   "disk",
					"io.bus": "usb",
					"source": isoPath,
				},
				// It seems that the lower the drive is, the logical name becomes closer to C.
				// It is necessary because my unattend xml file load virtio drivers only for D, E & F drives.
				// With this position virtio should be D.
				"virtio": {
					"type": "disk",
					// USB bus supported since incus v6.11
					"io.bus": "usb",
					"source": path.Join(downloadsDir, "virtio.iso"),
				},
			},
		},
	}

	op, err := c.CreateInstance(instance)
	if err != nil {
		return fmt.Errorf("instance creation failed %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("waiting operation failed: %w", err)
	}

	// Start the VM
	op, err = c.UpdateInstanceState(args.Name, api.InstanceStatePut{Action: "start", Timeout: -1}, "")
	if err != nil {
		return fmt.Errorf("instance update failed: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("waiting operation failed: %w", err)
	}

	// Send a key to QEMU to skip the "Press any key to boot from CD or DVD..." windows prompt
	time.Sleep(1 * time.Second)
	sendMonitorKeys("ret", monitorAddr, 30)

	q := make(chan string)
	listener, evt := eventHandler(c, args.Name, q)

	log.Debug("installing Windows Files")
	fmt.Print("\rStatus: [1/3] Installing Windows Files...           ")

	ev := timeout(q, 8*time.Minute)
	// Windows have taken more than 8 minutes for this step, there is something wrong
	if ev != "instance-restarted" {
		log.Debug("missing first restart")
		return fmt.Errorf("failed to complete the first install step")
	}

	log.Debug("configuring System Settings")
	fmt.Print("\rStatus: [2/3] Configuring System Settings...        ")

	ev = timeout(q, 4*time.Minute)
	// Windows have taken more than 4 minutes for this step, there is something wrong
	if ev != "instance-restarted" {
		log.Debug("missing second restart")
		return fmt.Errorf("failed to complete the second install step")
	}

	log.Debug("finishing Setup & Scripts")
	fmt.Print("\rStatus: [3/3] Finishing Setup & Scripts...          ")

	ev = timeout(q, 4*time.Minute)
	// Windows have taken more than 4 minutes for this step, there is something wrong
	if ev != "instance-shutdown" {
		log.Debug("missing Shutdown")
		return fmt.Errorf("failed to shutdown at the end of the install")
	}

	listener.RemoveHandler(evt)
	listener.Disconnect()

	log.Debug("update config to add bottle")
	cfg := config.Get()

	// Update the config
	cfg.Bottles = append(cfg.Bottles, config.Bottle{
		Name:     args.Name,
		Version:  "v2",
		Password: args.Password,
	})

	// Set the default bottle if not already set
	if cfg.DefaultBottle == "" {
		cfg.DefaultBottle = args.Name
	}

	removeDevice(c, args.Name, "software")
	removeDevice(c, args.Name, "autounattend")
	removeDevice(c, args.Name, "virtio")
	return nil
}

// wait on the channel some amount of time
func timeout(q <-chan string, d time.Duration) string {
	select {
	case ev := <-q:
		return ev
	case <-time.After(d):
		return ""
	}
}

func removeDevice(c incus.InstanceServer, vmName string, device string) {
	inst, etag, err := c.GetInstance(vmName)
	if err != nil {
		log.Fatal("failed to fetch instance for cleanup", "err", err)
	}

	delete(inst.Devices, device)

	op, err := c.UpdateInstance(vmName, inst.Writable(), etag)
	if err != nil {
		log.Fatal("cleanup failed", "err", err)
	}
	op.Wait()
}

func eventHandler(c incus.InstanceServer, vmName string, q chan<- string) (*incus.EventListener, *incus.EventTarget) {
	listener, err := c.GetEvents()
	if err != nil {
		log.Fatal("failed to connect to event stream", "err", err)
	}

	log.Debug("connected to event stream", "vm", vmName)

	countRestart := 0

	// Lifecycle events cover instance creation, start, stop, etc.
	evt, err := listener.AddHandler([]string{"lifecycle"}, func(event api.Event) {
		// Metadata in lifecycle events is a JSON raw message
		var lifecycle api.EventLifecycle
		err := json.Unmarshal(event.Metadata, &lifecycle)
		if err != nil {
			return
		}

		// Filter events for our specific instance
		// The Source for instance events is typically "/1.0/instances/<name>"
		targetSource := fmt.Sprintf("/1.0/instances/%s", vmName)
		if lifecycle.Source == targetSource {
			log.Debug("event", "time", time.Now().Format("15:04:05"), "action", lifecycle.Action)

			if lifecycle.Action == "instance-restarted" {
				countRestart++
			}

			if lifecycle.Action == "instance-restarted" && countRestart == 1 {
				removeDevice(c, vmName, "install")
				log.Debug("install ISO removed.")
			}

			if lifecycle.Action != "instance-updated" {
				q <- lifecycle.Action
			}
		}

	})

	if err != nil {
		log.Fatal("failed to add event handler", "err", err)
	}

	return listener, evt
}
