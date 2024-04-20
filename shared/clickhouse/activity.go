package clickhouse

import (
	"database/sql"
	"fmt"
)

type Activity struct {
	Hash       uint32
	ActivityId uint8
	VersionId  uint8
}

type InsertActivityStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareActivity(tx *sql.Tx) (*InsertActivityStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO activity (hash, activity_id, version_id) VALUES (?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for activity: %s", err)
	} else {
		return &InsertActivityStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertActivityStatement) Exec(data *Activity) (sql.Result, error) {
	result, err := s.stmt.Exec(data.Hash, data.ActivityId, data.VersionId)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for activity %d: %s", data.Hash, err)
	} else {
		return result, nil
	}
}

func (s *InsertActivityStatement) Close() error {
	return s.stmt.Close()
}
