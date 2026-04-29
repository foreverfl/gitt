package client

import (
	"fmt"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/paths"
)

// SqliteTest asks the daemon to run its scratch-table self-test and returns
// the human-readable summary line. Used by `gitt sqlite` to confirm the
// daemon's database connection is healthy.
func SqliteTest() (string, error) {
	sockpath, err := paths.SockPath()
	if err != nil {
		return "", err
	}
	response, err := Call(sockpath, daemon.Request{Op: daemon.OpSqliteTest})
	if err != nil {
		return "", err
	}
	if !response.OK {
		return "", fmt.Errorf("%s", response.Error)
	}
	message, _ := response.Data["message"].(string)
	return message, nil
}
