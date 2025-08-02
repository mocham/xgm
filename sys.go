package main
import (
    "os"
	"os/exec"
	"io/ioutil"
    "path/filepath"
    "strings"
	"io"
	_ "embed"
    "syscall"
	"net"
	"github.com/mocham/xgw"
	"sync"
)
var (
	mouseProcess *os.Process
	//go:embed CPlugins/src/mouse_monitor
	monitorData []byte
    rxPrev, txPrev int
    logFile, rxFile, txFile, tempFile, batteryNowFile, batteryFullFile *os.File
	barThreadSafe bool = true
	statsLock sync.Mutex
	barCache [50]uint32
	socketPath = "/tmp/bar.sock"
)

func mouseMonitor() {
	xgw.LogAndExit(cleanup, os.WriteFile("/tmp/monitor_mouse", monitorData, 0755))
	cmd := exec.Command("/tmp/monitor_mouse")
	cmd.SysProcAttr, cmd.Stdout, cmd.Stderr = &syscall.SysProcAttr{Setsid: true}, os.Stdout, os.Stderr
	xgw.LogAndExit(cleanup, cmd.Start())
	mouseProcess = cmd.Process
}

func getBrightness() (deviceID string, val int) {
	if files, err := ioutil.ReadDir("/sys/class/backlight"); err != nil || len(files) > 0 { deviceID = files[0].Name() }
	if data, err := os.ReadFile(filepath.Join("/sys/class/backlight", deviceID, "brightness")); err == nil { val = xgw.ParseInt(strings.TrimSpace(string(data))) }
	return deviceID, val
}

func brightChange(delta int) {
	deviceID, curr := getBrightness()
	if curr + delta < 0 { curr = 0 } else { curr += delta }
	cmd := exec.Command("dbus-send", "--system", "--dest=org.freedesktop.login1",
		"--type=method_call", "--print-reply", "/org/freedesktop/login1/session/auto",
		"org.freedesktop.login1.Session.SetBrightness",
		"string:backlight", "string:" + deviceID, "uint32:" + xgw.FmtInt(curr))
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	cmd.Run()
}

func getNetworkInterface(prefix string) string {
	files, _ := filepath.Glob("/sys/class/net/*")
    for _, f := range files { if iface := filepath.Base(f); strings.HasPrefix(iface, strings.TrimPrefix(prefix, "^")) { return iface } }
    return ""
}

func readIntFromFile(file *os.File) int {
    if file == nil { return 0 }
    if _, err := file.Seek(0, io.SeekStart); err != nil { return 0 }
    data, err := io.ReadAll(file)
    if err != nil { return 0 }
    return xgw.ParseInt(strings.TrimSpace(string(data)))
}

func ipcSend(socketPath string, data []byte) []byte{
    data = append(data, 0)
    conn, err := net.Dial("unix", socketPath)
	xgw.LogAndExit(cleanup, err)
    defer conn.Close()
    _, err = conn.Write([]byte(data))
	xgw.LogAndExit(cleanup, err)
    response, err := io.ReadAll(conn)
	xgw.LogAndExit(cleanup, err)
    return response
}

func runDetachedWithWorkDir(workdir string, args ...string) {
    if !xgw.IsDir(workdir) { return }
    cmd := exec.Command(args[0], args[1:]...)
    cmd.Dir, cmd.SysProcAttr, cmd.Stdout, cmd.Stderr = workdir, &syscall.SysProcAttr{Setsid: true, Foreground: false, Setctty: false}, io.Discard, io.Discard
    cmd.Start()
    go cmd.Wait()
}
