package v2

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"

	"github.com/plus3it/gorecurcopy"
)

type NewV2Argument struct {
	Name     string
	Ram      uint
	Disk     uint
	Cpus     uint
	Password string
	Username string
}

type windowsLocales struct {
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

func detectLocale() (windowsLocales, error) {
	out, err := exec.Command("localectl").Output()
	if err != nil {
		return windowsLocales{}, err
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
	return windowsLocales{
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
		return path.Join(wd, "assets", "v2", "autounattend.xml"), nil
	}

	item, err := cache.Get("v2/autounattend.xml")
	return item.Path, err
}

// Copy the assets for creating a autounattend iso file to a temp dir
func copyUnattendAssetsToDir(d string) error {
	log.Debug("temp directory", "dir", d)

	version := config.GetVersion()
	if version.Version == "dev" {
		wd, _ := os.Getwd()
		gorecurcopy.CopyDirectory(path.Join(wd, "assets", "v2"), d)
	} else {
		cache.CopyFromCache(d, []string{"v2/scripts/oobe.ps1", "v2/scripts/specialize.ps1", "v2/sripts/pe.cmd"})
	}

	return nil
}

func createAutounattendISO(args NewV2Argument) (string, error) {
	tmpDir, err := os.MkdirTemp("", "lsw-autounattend")
	if err != nil {
		return "", err
	}

	isoPath := path.Join(tmpDir, "autounattend.iso")

	err = copyUnattendAssetsToDir(tmpDir)
	if err != nil {
		utils.Panic("error when copying assets to temporary directory", err)
	}

	locale, err := detectLocale()
	if err != nil {
		utils.Panic("error when calling localectl", err)
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
			log.Warn("failed to send key to QEMU monitor", "err", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func New(arch string, args NewV2Argument) error {
	Init()

	log.Info("creating new bottle (v2 backend)", "name", args.Name)

	if arch != "amd64" {
		utils.Panic("not supported architecture")
	}

	// Create the autounattend iso
	isoPath, err := createAutounattendISO(args)
	if err != nil {
		return err
	}
	// defer os.RemoveAll(path.Dir(isoPath))

	c, err := incusClient()
	if err != nil {
		return fmt.Errorf("failed to connect to incus socket: %w", err)
	}

	cacheDir, err := cache.GetCacheDir()
	if err != nil {
		utils.Panic("cannot get cache dir")
	}

	virtioIso, err := cache.Get("v2/virtio.iso")
	if err != nil {
		return err
	}

	softwareIso, err := cache.Get("v2/software.iso")
	if err != nil {
		return err
	}

	port, err := getFreePort()
	if err != nil {
		return err
	}
	monitorAddr := fmt.Sprintf("127.0.0.1:%d", port)

	diskSize := fmt.Sprintf("%dGiB", args.Disk)
	ramSize := fmt.Sprintf("%dGiB", args.Ram)
	cpu := strconv.FormatUint(uint64(3), 10)

	const qemuArgs = "-device intel-hda -device hda-duplex -audio spice"
	qemuTcp := "-monitor tcp:" + monitorAddr + ",server,nowait"

	instance := api.InstancesPost{
		Name: args.Name,
		Type: api.InstanceTypeVM,
		Source: api.InstanceSource{
			Type: "none",
		},
		InstancePut: api.InstancePut{
			Config: map[string]string{
				"image.os":      "Windows",
				"limits.cpu":    cpu,
				"limits.memory": ramSize,
				// Expose QEMU monitor via TCP to send the "any key" bypass
				"raw.qemu": qemuArgs + " " + qemuTcp,
				// Fix to avoid
				// ❯ sudo cat /var/log/incus/win-1/qemu.log
				// qemu-system-x86_64: -device hda-duplex: no default audio driver available
				// Perhaps you wanted to use -audio or set audiodev=qemu_spice-audiodev?
				// Might need to investigate later
				"raw.qemu.conf": "[audiodev \"qemu_spice-audiodev\"]\ndriver = \"none\"",
			},
			Devices: map[string]map[string]string{
				"root": {
					"type": "disk",
					"pool": "default",
					"path": "/",
					"size": diskSize,
					// "io.bus": "nvme",
				},
				"vtpm": {"type": "tpm", "path": "/dev/tpm0"},
				"eth0": {
					"type":    "nic",
					"network": "incusbr0",
				},
				// It seems that the lower the drive is, the logical name becomes closer to C.
				// It is necessary because my unattend xml file load virtio drivers only for D, E & F drives.
				// With this position virtio should be D.
				"virtio": {
					"type": "disk",
					// USB bus supported since incus v6.11
					"io.bus": "usb",
					"source": virtioIso.Path,
				},
				"autounattend": {
					"type":   "disk",
					"io.bus": "usb",
					"source": isoPath,
				},
				"install": {
					"type":          "disk",
					"source":        path.Join(cacheDir, "windows-server.iso"),
					"boot.priority": "10",
				},
			},
		},
	}

	// Devices that are added after windows kernel is installed
	otherDevices := map[string]map[string]string{
		// ISO containing SSH and winfsp
		"software": {
			"type":   "disk",
			"io.bus": "usb",
			"source": softwareIso.Path,
		},
		"agent": {
			"type":   "disk",
			"io.bus": "usb",
			"source": "agent:config",
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
	listener, evt := eventHandler(c, args.Name, q, otherDevices)

	log.Info("installing Windows files")
	progressCallback := utils.GetProgressCallback()
	if progressCallback != nil {
		progressCallback("Installing Windows Files...", utils.ProgressStart)
	}

	ev := timeout(q, 8*time.Minute)
	// Windows have taken more than 8 minutes for this step, there is something wrong
	if ev != "instance-restarted" {
		if progressCallback != nil {
			progressCallback("Failed to complete first install step", utils.ProgressError)
		}
		log.Warn("VM did not restart within expected time during Windows installation")
		return fmt.Errorf("failed to complete the first install step")
	}

	log.Info("configuring system settings")
	if progressCallback != nil {
		progressCallback("Configuring System Settings...", utils.ProgressUpdate)
	}

	ev = timeout(q, 4*time.Minute)
	// Windows have taken more than 4 minutes for this step, there is something wrong
	if ev != "instance-restarted" {
		if progressCallback != nil {
			progressCallback("Failed to complete system settings configuration", utils.ProgressError)
		}
		log.Warn("VM did not restart within expected time during system settings configuration")
		return fmt.Errorf("failed to complete the second install step")
	}

	log.Info("finishing setup and scripts")

	if progressCallback != nil {
		progressCallback("Finishing Setup & Scripts...", utils.ProgressUpdate)
	}

	ev = timeout(q, 5*time.Minute)
	// Windows have taken more than 4 minutes for this step, there is something wrong
	if ev != "instance-shutdown" {
		if progressCallback != nil {
			progressCallback("Failed to shutdown at end of install", utils.ProgressError)
		}
		log.Warn("VM did not shut down within expected time during Windows installation")
		return fmt.Errorf("failed to shutdown at the end of the install")
	}

	listener.RemoveHandler(evt)
	listener.Disconnect()

	log.Info("updating config to add new bottle")

	// Update the config
	config.Get().AddBottle(config.Bottle{
		Name:     args.Name,
		Version:  "v2",
		Password: args.Password,
	})

	err = removeDevices(c, args.Name, []string{"software", "autounattend", "virtio"})
	if err != nil {
		if progressCallback != nil {
			progressCallback("Failed to remove installation devices", utils.ProgressError)
		}
		utils.Panic("failed to remove installation devices", err)
	}

	err = updateInstance(c, args.Name, func(inst *api.Instance) error {
		inst.Config["raw.qemu"] = qemuArgs
		return nil
	})
	if err != nil {
		if progressCallback != nil {
			progressCallback("Failed to update raw.qemu config", utils.ProgressError)
		}
		utils.Panic("failed to update raw.qemu config", err)
	}

	if progressCallback != nil {
		progressCallback("Bottle created successfully", utils.ProgressDone)
	}

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

func eventHandler(c incus.InstanceServer, vmName string, q chan<- string, devicesToAdd map[string]map[string]string) (*incus.EventListener, *incus.EventTarget) {
	listener, err := c.GetEvents()
	if err != nil {
		utils.Panic("failed to connect to event stream", err)
	}

	log.Info("connected to Incus event stream", "vm", vmName)

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
				err := removeDevices(c, vmName, []string{"install"})
				if err != nil {
					log.Error("failed to remove install ISO", "err", err)
					// Decide how to handle this critical error. For now, log and continue, but this might need a more robust error path.
				}
				err = addDevices(c, vmName, devicesToAdd)
				if err != nil {
					log.Error("failed to add devices", "err", err)
					// Decide how to handle this critical error.
				}
				log.Info("install ISO removed.")
			}

			if lifecycle.Action == "instance-restarted" || lifecycle.Action == "instance-shutdown" {
				q <- lifecycle.Action
			}
		}

	})

	if err != nil {
		utils.Panic("failed to add event handler", err)
	}

	return listener, evt
}
