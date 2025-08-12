package main
import (
	"io"
	"fmt"
	"os"
	"net"
	"path/filepath"
	"strings"
)
const socketPath = "/tmp/bar.sock"
func send(text string) { sendRaw([]byte(text)) }
func sendRaw(data []byte) {
    data = append(data, 0)
    conn, err := net.Dial("unix", socketPath)
	if err != nil { return }
    defer conn.Close()
    _, err = conn.Write([]byte(data))
	if err != nil { return }
    response, _ := io.ReadAll(conn)
	fmt.Printf(string(response))
}

func main() {
	args := os.Args
	if len(args) < 2 { return }
	switch args[1] {
	case "ls", "zip", "unzip", "open", "cache": if len(args) >= 3 {
		fn := args[2]
		if len(fn) < 1 { return }
		if fn[0] != '/' { fn = filepath.Join(os.Getenv("PWD"), fn)}
		send(fmt.Sprintf("%s\n%s", strings.ToUpper(args[1]), fn))
	}
	case "purge": if len(args) >= 3 {
		if args[2] == "vm" { printPurgable(vmKeepPkgs) }
		if args[2] == "laptop" { printPurgable(laptopKeepPkgs) }
	}
	case "git": if len(args) >= 3 {
		if args[2] == "sync" && len(args) >= 5 { gitSync(args[3], args[4]) }
		if args[2] != "sync" { sandBoxedGit("", args[2:]...) }
	}
	case "jup": sandboxedJupyter(args[1:]...)
	case "meminfo": printMeminfo(args[1:]...)
	case "eopen": if len(args) >= 3 {send(fmt.Sprintf("EXTERNAL\n%s", args[2])) }
	case "msg": if data, err := io.ReadAll(os.Stdin); err == nil { sendRaw(data) }
	case "args": send(strings.Join(args[2:], "\n"))
	case "sel": if data, err := io.ReadAll(os.Stdin); err == nil { send("SETSEL\n" + strings.Replace(string(data), "\t", "    ", -1)) }
	}
}
