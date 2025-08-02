package main
import (
	"os/exec"
	"os"
	"path/filepath"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil { return }
	cmd.Wait()
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
