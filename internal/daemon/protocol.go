package daemon

// Op is the RPC method name carried by Request.
type Op string

const (
	OpPing       Op = "ping"
	OpShutdown   Op = "shutdown"
	OpRegister   Op = "register"    // create worktree + allocate ports
	OpRelease    Op = "release"     // delete worktree + free ports
	OpSqliteTest Op = "sqlite_test" // create+insert+select+drop a scratch table
)

type Request struct {
	Op   Op             `json:"op"`
	Args map[string]any `json:"args,omitempty"`
}

type Response struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data,omitempty"`
	Error string         `json:"error,omitempty"`
}
