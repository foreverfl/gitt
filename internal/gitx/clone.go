package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneBare runs `git clone --bare <url> <dest>`, streaming git's progress to
// the caller's stdout/stderr. It also installs the standard fetch refspec so
// `refs/remotes/origin/*` populates on subsequent fetches — `git clone --bare`
// omits this by default, which breaks ahead/behind tracking in editors and
// `--force-with-lease` on push.
func CloneBare(url, dest string) error {
	cmd := exec.Command("git", "clone", "--bare", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone --bare: %w", err)
	}
	cfg := exec.Command("git", "--git-dir", dest, "config",
		"remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if out, err := cfg.CombinedOutput(); err != nil {
		return fmt.Errorf("set remote.origin.fetch: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fetch := exec.Command("git", "--git-dir", dest, "fetch", "origin")
	fetch.Stdout = os.Stdout
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return fmt.Errorf("initial fetch: %w", err)
	}
	return nil
}

// HeadBranchOf returns the symbolic-ref short name HEAD points at inside the
// given git directory (typically a bare repo). After `git clone --bare`, HEAD
// mirrors the remote's default branch, so this is gitt's default-branch probe.
func HeadBranchOf(gitDir string) (string, error) {
	out, err := exec.Command("git", "--git-dir", gitDir, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git symbolic-ref HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DeriveCloneDir returns the basename git itself would pick as the clone target
// directory: trailing slashes stripped, optional .git suffix removed, then the
// segment after the last '/' or ':'.
//
//	https://github.com/foo/bar.git → "bar"
//	git@github.com:foo/bar.git    → "bar"
//	/local/path/to/repo.git/      → "repo"
func DeriveCloneDir(url string) string {
	s := strings.TrimRight(url, "/")
	s = strings.TrimSuffix(s, ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	return s
}