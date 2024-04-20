package clickhouse

import (
	"database/sql"
	"fmt"
)

type InstancePlayer struct {
	InstanceId        int64
	MembershipId      int64
	Completed         bool
	TimePlayedSeconds int
	Sherpas           int
	IsFirstClear      bool
}

type InsertInstancePlayerStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareInstancePlayer(tx *sql.Tx) (*InsertInstancePlayerStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO instance_player (instance_id, membership_id, completed, time_played_seconds, sherpas, is_first_clear) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for instance_player: %s", err)
	} else {
		return &InsertInstancePlayerStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertInstancePlayerStatement) Exec(data *InstancePlayer) (sql.Result, error) {
	result, err := s.stmt.Exec(data.InstanceId, data.MembershipId, data.Completed, data.TimePlayedSeconds, data.Sherpas, data.IsFirstClear)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for instance_player %d: %s", data.InstanceId, err)
	} else {
		return result, nil
	}
}

func (s *InsertInstancePlayerStatement) Close() error {
	return s.stmt.Close()
}
