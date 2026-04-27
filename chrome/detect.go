package chrome

import (
	"os"
	"path/filepath"
	"runtime"
)

// candidates is the ordered list of browser binary names to search.
// More specific names (versioned, platform-specific) come first.
var candidates = []string{
	"chromium",
	"chromium-browser",
	"google-chrome",
	"google-chrome-stable",
	"google-chrome-beta",
	"chrome",
}

// platformCandidates adds OS-specific well-known installation paths.
func platformCandidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "windows":
		roots := []string{
			os.Getenv("PROGRAMFILES"),
			os.Getenv("PROGRAMFILES(X86)"),
			os.Getenv("LOCALAPPDATA"),
		}
		var paths []string
		for _, root := range roots {
			if root == "" {
				continue
			}
			paths = append(paths,
				filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(root, "Chromium", "Application", "chrome.exe"),
			)
		}
		return paths
	default:
		return nil
	}
}
