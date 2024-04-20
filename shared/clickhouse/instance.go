package clickhouse

import (
	"database/sql"
	"fmt"
	"time"
)

type Instance struct {
	InstanceId                 int64
	Hash                       uint32
	Completed                  bool
	PlayerCount                int
	Fresh, Flawless            uint8
	DateStarted, DateCompleted time.Time
	PlatformType               uint16
	Duration, Score            int
}

type InsertInstanceStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareInstance(tx *sql.Tx) (*InsertInstanceStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO instance (instance_id, hash, completed, player_count, fresh, flawless, date_started, date_completed, platform_type, duration, score) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for instance: %s", err)
	} else {
		return &InsertInstanceStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertInstanceStatement) Exec(data *Instance) (sql.Result, error) {
	result, err := s.stmt.Exec(data.InstanceId, data.Hash, data.Completed, data.PlayerCount, data.Fresh, data.Flawless, data.DateStarted, data.DateCompleted, data.PlatformType, data.Duration, data.Score)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for instance %d: %s", data.InstanceId, err)
	} else {
		return result, nil
	}
}

func (s *InsertInstanceStatement) Close() error {
	return s.stmt.Close()
}
