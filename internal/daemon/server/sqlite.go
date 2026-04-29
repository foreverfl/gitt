package server

import "github.com/foreverfl/gitt/internal/daemon"

// handleSqliteTest runs the store's scratch-table self-test and returns the
// summary line in Response.Data["message"]. cmd/sqlite uses this to confirm
// the daemon's database connection is healthy end-to-end.
func (s *server) handleSqliteTest(_ daemon.Request) daemon.Response {
	summary, err := s.store.Test()
	if err != nil {
		return daemon.Response{OK: false, Error: err.Error()}
	}
	return daemon.Response{OK: true, Data: map[string]any{"message": summary}}
}
