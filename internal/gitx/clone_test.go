package gitx

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneBare_InstallsFetchRefspecAndPopulatesRemoteRefs(t *testing.T) {
	source := setupNormalRepo(t)
	dest := filepath.Join(t.TempDir(), "dest.git")

	if err := CloneBare(source, dest); err != nil {
		t.Fatalf("CloneBare: %v", err)
	}

	out, err := exec.Command("git", "--git-dir", dest, "config", "remote.origin.fetch").Output()
	if err != nil {
		t.Fatalf("read remote.origin.fetch: %v", err)
	}
	got := strings.TrimSpace(string(out))
	want := "+refs/heads/*:refs/remotes/origin/*"
	if got != want {
		t.Errorf("remote.origin.fetch = %q, want %q", got, want)
	}

	refOut, err := exec.Command("git", "--git-dir", dest, "for-each-ref", "--format=%(refname)", "refs/remotes/origin/").Output()
	if err != nil {
		t.Fatalf("for-each-ref: %v", err)
	}
	if !strings.Contains(string(refOut), "refs/remotes/origin/main") {
		t.Errorf("refs/remotes/origin/main not present after CloneBare; refs:\n%s", refOut)
	}
}

func TestDeriveCloneDir(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://github.com/foo/bar.git", "bar"},
		{"https://github.com/foo/bar", "bar"},
		{"https://github.com/foo/bar/", "bar"},
		{"git@github.com:foo/bar.git", "bar"},
		{"git@github.com:foo/bar", "bar"},
		{"/local/path/to/repo.git/", "repo"},
		{"/local/path/to/repo", "repo"},
		{"repo", "repo"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := DeriveCloneDir(tc.in); got != tc.want {
			t.Errorf("DeriveCloneDir(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}