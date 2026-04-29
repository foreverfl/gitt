// Package daemon defines the wire protocol shared by the gitt CLI (client
// side, internal/daemon/client) and the long-lived daemon process (server
// side, internal/daemon/server). The envelope (Op enum, Request, Response)
// lives here; per-op payload structs are in messages.go.
package daemon

import "encoding/json"

// Op is the RPC method name carried by Request.
type Op string

const (
	OpPing             Op = "ping"
	OpShutdown         Op = "shutdown"
	OpRegisterWorktree Op = "register_worktree" // persist a worktree row
	OpListWorktrees    Op = "list_worktrees"    // read all worktree rows
	OpRenameWorktree   Op = "rename_worktree"   // rename branch + move folder + update row
	OpRelease          Op = "release"           // delete worktree row
	OpSqliteTest       Op = "sqlite_test"       // create+insert+select+drop a scratch table
)

// Request is the wire frame the client sends to the daemon. Args is an
// opaque JSON blob whose shape is determined by Op; see messages.go for the
// typed views (RegisterWorktreeArgs, RenameWorktreeArgs, ...).
type Request struct {
	Op   Op              `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
}

// Response is the wire frame the daemon sends back. Data, like Args, is a
// JSON blob whose typed view depends on Op (WorktreeData, ListWorktreesData,
// SqliteTestData, ...). Empty Data is fine for ops that only signal success.
type Response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}
