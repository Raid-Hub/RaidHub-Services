package clickhouse

import (
	"database/sql"
	"fmt"
)

type ActivityDefinition struct {
	Id       int64
	Name     string
	IsSunset bool
	IsRaid   bool
}

type InsertActivityDefinitionStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareActivityDefinition(tx *sql.Tx) (*InsertActivityDefinitionStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO activity_definition (id, name, is_sunset, is_raid) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for activity_definition: %s", err)
	} else {
		return &InsertActivityDefinitionStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertActivityDefinitionStatement) Exec(data *ActivityDefinition) (sql.Result, error) {
	result, err := s.stmt.Exec(data.Id, data.Name, data.IsSunset, data.IsRaid)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for activity_definition %d: %s", data.Id, err)
	} else {
		return result, nil
	}
}

func (s *InsertActivityDefinitionStatement) Close() error {
	return s.stmt.Close()
}
