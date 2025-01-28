package youtubelive

import (
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = "open"
		args = []string{url}
	case "windows":
		// Windows
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
