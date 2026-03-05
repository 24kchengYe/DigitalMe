package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CaptureScreen takes a screenshot of the primary display.
// Currently only supports Windows via PowerShell.
func CaptureScreen() ([]byte, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("screenshot not supported on %s", runtime.GOOS)
	}

	tmpFile := filepath.Join(os.TempDir(), "digitalme_screenshot.png")
	script := "Add-Type -AssemblyName System.Windows.Forms;Add-Type -AssemblyName System.Drawing;" +
		"$b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds;" +
		"$bmp=New-Object System.Drawing.Bitmap($b.Width,$b.Height);" +
		"$g=[System.Drawing.Graphics]::FromImage($bmp);" +
		"$g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size);" +
		"$bmp.Save('" + tmpFile + "');" +
		"$g.Dispose();$bmp.Dispose()"

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("screenshot capture: %w", err)
	}
	defer os.Remove(tmpFile)
	return os.ReadFile(tmpFile)
}
