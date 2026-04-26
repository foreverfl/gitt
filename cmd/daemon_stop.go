package cmd

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/foreverfl/doctree/internal/daemon"
)

const daemonStopTimeout = 3 * time.Second

// shutdownDaemon stops the daemon if it is running, prints status, and cleans
// up the sock/pid files. Returns nil when the daemon wasn't running or was
// stopped successfully; non-nil only when the SIGTERM fallback also failed.
// Shared by `doctree off` and `doctree uninstall`.
func shutdownDaemon() error {
	sockpath, err := sockPath()
	if err != nil {
		return err
	}
	pidpath, err := pidPath()
	if err != nil {
		return err
	}

	pid, hasPid := readPid(pidpath)
	if !hasPid || !processAlive(pid) {
		_ = os.Remove(sockpath)
		_ = os.Remove(pidpath)
		fmt.Println("doctree daemon not running")
		return nil
	}

	_, callErr := daemon.Call(sockpath, daemon.Request{Op: daemon.OpShutdown})

	if waitExit(pid, daemonStopTimeout) {
		_ = os.Remove(sockpath)
		_ = os.Remove(pidpath)
		fmt.Printf("doctree daemon stopped (pid=%d)\n", pid)
		return nil
	}

	if callErr != nil && !errors.Is(callErr, daemon.ErrNotRunning) {
		fmt.Fprintf(os.Stderr, "rpc shutdown failed: %v; sending SIGTERM\n", callErr)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("sigterm pid=%d: %w", pid, err)
	}
	if !waitExit(pid, daemonStopTimeout) {
		return fmt.Errorf("daemon did not exit after SIGTERM (pid=%d)", pid)
	}
	_ = os.Remove(sockpath)
	_ = os.Remove(pidpath)
	fmt.Printf("doctree daemon stopped via SIGTERM (pid=%d)\n", pid)
	return nil
}
