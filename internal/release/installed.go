package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/foreverfl/gitt/internal/paths"
)

// versionFile is the on-disk record of the currently installed gitt
// version. install.sh creates it; update.go rewrites it via MarkInstalled.
const versionFile = "VERSION"

// versionPath resolves ~/.gitt/VERSION. Kept private because the file is an
// internal handshake between install.sh and the binary; callers should go
// through Installed / MarkInstalled instead of touching the path directly.
func versionPath() (string, error) {
	dir, err := paths.RuntimeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, versionFile), nil
}

// Installed returns the version string recorded by install.sh in
// ~/.gitt/VERSION, or "" when the file is missing or unreadable. cmd/version
// surfaces "" as "unknown (not installed via install.sh)".
func Installed() string {
	path, err := versionPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// MarkInstalled records version as the current installed gitt version,
// rewriting ~/.gitt/VERSION. Called by `gitt update` after the new binary
// is in place, so the next Installed() call reflects the upgrade.
func MarkInstalled(version string) error {
	path, err := versionPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(version+"\n"), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", versionFile, err)
	}
	return nil
}
