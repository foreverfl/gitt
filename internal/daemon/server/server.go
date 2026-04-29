// Package server is the long-lived daemon process: it listens on the unix
// socket, dispatches Op-keyed handlers, and owns the SQLite store. It is
// only ever started by `gitt daemon-run`, which `gitt on` fork-execs into.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/store"
)

type handler func(req daemon.Request) daemon.Response

type server struct {
	listener     net.Listener
	store        *store.Store
	handlers     map[daemon.Op]handler
	shutdown     chan struct{}
	shutdownOnce sync.Once
	wg           sync.WaitGroup
}

// Run starts the daemon: opens the SQLite store at dbPath and listens on
// sockPath. Blocks until OpShutdown / SIGTERM / SIGINT.
func Run(sockPath, dbPath string) error {
	sqliteStore, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer sqliteStore.Close()

	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear stale sock: %w", err)
	}

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen %s: %w", sockPath, err)
	}

	srv := &server{
		listener: listener,
		store:    sqliteStore,
		shutdown: make(chan struct{}),
	}
	srv.handlers = map[daemon.Op]handler{
		daemon.OpPing: func(_ daemon.Request) daemon.Response { return daemon.Response{OK: true} },
		daemon.OpShutdown: func(_ daemon.Request) daemon.Response {
			// Close listener first so accept loop exits, then signal Run to
			// finish. Done in a goroutine so this handler can still return its
			// response on the same connection before the conn is torn down.
			go srv.shutdownOnce.Do(func() { close(srv.shutdown) })
			return daemon.Response{OK: true}
		},
		daemon.OpSqliteTest:       srv.handleSqliteTest,
		daemon.OpRegisterWorktree: srv.handleRegisterWorktree,
		daemon.OpListWorktrees:    srv.handleListWorktrees,
		daemon.OpRenameWorktree:   srv.handleRenameWorktree,
		daemon.OpRelease:          srv.handleRelease,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go srv.acceptLoop()

	select {
	case <-srv.shutdown:
	case <-sigCh:
	}

	_ = listener.Close()
	srv.wg.Wait()
	_ = os.Remove(sockPath)
	return nil
}

func (s *server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			fmt.Fprintln(os.Stderr, "gitt: accept:", err)
			continue
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(conn)
		}()
	}
}

func (s *server) handleConn(conn net.Conn) {
	defer conn.Close()

	var req daemon.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(daemon.Response{OK: false, Error: "decode: " + err.Error()})
		return
	}

	h, ok := s.handlers[req.Op]
	if !ok {
		_ = json.NewEncoder(conn).Encode(daemon.Response{OK: false, Error: "unknown op: " + string(req.Op)})
		return
	}
	_ = json.NewEncoder(conn).Encode(h(req))
}
