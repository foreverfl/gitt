package cmd

import (
	"os"
	"strings"

	"github.com/foreverfl/doctree/internal/paths"
)

// installedVersion returns the version recorded by install.sh in
// ~/.doctree/VERSION, or "" if not recorded or unreadable.
func installedVersion() string {
	path, err := paths.VersionPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}