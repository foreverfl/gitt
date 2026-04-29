// Package client holds the caller-side helpers that talk to the gitt daemon
// over its unix socket. It is split from internal/daemon (which only carries
// the wire protocol types) so consumers like cmd/* import only this package.
package client

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/foreverfl/gitt/internal/daemon"
)

// ErrNotRunning is returned when no daemon socket is reachable at the given path.
// cmd/* surfaces this as the "run gitt on first" hint.
var ErrNotRunning = errors.New("gitt daemon not running")

const (
	dialTimeout = 2 * time.Second
	rwTimeout   = 5 * time.Second
)

// Ping checks whether a daemon is reachable at sockPath by issuing OpPing.
// Returns ErrNotRunning when the socket is missing or the dial is refused.
func Ping(sockPath string) error {
	if _, err := os.Stat(sockPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotRunning
		}
		return err
	}
	resp, err := Call(sockPath, daemon.Request{Op: daemon.OpPing})
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New("daemon ping rejected: " + resp.Error)
	}
	return nil
}

// Call sends a single Request to the daemon and returns its Response.
// ECONNREFUSED / ENOENT on dial map to ErrNotRunning so callers can give the
// same "run gitt on first" hint regardless of which failure mode hit.
func Call(sockPath string, req daemon.Request) (daemon.Response, error) {
	conn, err := net.DialTimeout("unix", sockPath, dialTimeout)
	if err != nil {
		if isNotRunning(err) {
			return daemon.Response{}, ErrNotRunning
		}
		return daemon.Response{}, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(rwTimeout)); err != nil {
		return daemon.Response{}, err
	}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return daemon.Response{}, err
	}
	var resp daemon.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return daemon.Response{}, err
	}
	return resp, nil
}

func isNotRunning(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) {
		return true
	}
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		return errors.Is(sysErr.Err, syscall.ECONNREFUSED) || errors.Is(sysErr.Err, syscall.ENOENT)
	}
	return false
}
