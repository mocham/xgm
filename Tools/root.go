package main
import (
	"io"
	"os/exec"
	"os"
	"path/filepath"
	"strings"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil { return }
	cmd.Wait()
}
func isVM() bool {
	vmIndicators := []string{
		"/sys/class/dmi/id/product_name",    // Typically contains "VMware" for VMware VMs
		"/sys/class/dmi/id/sys_vendor",     // Often "VMware, Inc."
		"/sys/bus/pci/devices/0000:00:0f.0", // VMware SVGA II PCI device
	}
	for _, path := range vmIndicators {
		if content, err := os.ReadFile(path); err == nil {
			if strings.Contains(strings.ToLower(string(content)), "vmware") {
				return true
			}
		}
	}
	if cpuinfo, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		if strings.Contains(strings.ToLower(string(cpuinfo)), "hypervisor") &&
			strings.Contains(strings.ToLower(string(cpuinfo)), "vmware") {
			return true
		}
	}
	return false
}
func resizeGuest() {
	cmd := exec.Command("bash", "-c", `MODE_NAME="HRes"
OUTPUT=$(xrandr | grep " connected" | cut -d' ' -f1)
MODELINE='2850x1450_60.00  347.25  2850 3064 3376 3902  1450 1453 1463 1505 -hsync +vsync'
xrandr --newmode "$MODE_NAME" 347.25  2850 3064 3376 3902  1450 1453 1463 1505 -hsync +vsync
xrandr --addmode "$OUTPUT" "$MODE_NAME"
xrandr --output "$OUTPUT" --mode "$MODE_NAME"`)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	(cmd.Run())
}
func main() {
	X, err := xgb.NewConn()
	if err != nil { return }
	defer X.Close()
	root := xproto.Setup(X).DefaultScreen(X).Root
	// Grab key combination: Mod1+Shift+B
	if xproto.GrabKeyChecked(X, true, root, 72, 56, xproto.GrabModeAsync, xproto.GrabModeAsync).Check() != nil { return }
	// Grab key combination: Mod4+XF86Calculator
	if xproto.GrabKeyChecked(X, true, root, 64, 148, xproto.GrabModeAsync, xproto.GrabModeAsync).Check() != nil { return }
	path := filepath.Join(os.Getenv("HOME"), "Bar", "Go", "Bin", "wm")
	if isVM() { resizeGuest() }
	run("setsid", path)
	for {
		ev, err := X.WaitForEvent()
		if err != nil { continue }
        switch event := ev.(type) {
        case xproto.KeyPressEvent:
            switch int(event.State) * 10000 + int(event.Detail) {
            case 720056: run("setsid", path)
            case 640148: run("sudo", "chvt", "2")
            }
		}
	}
}
