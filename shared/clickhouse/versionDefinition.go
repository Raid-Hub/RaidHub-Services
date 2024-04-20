package clickhouse

import (
	"database/sql"
	"fmt"
)

type VersionDefinition struct {
	Id   int64
	Name string
}

type InsertVersionDefinitionStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareVersionDefinition(tx *sql.Tx) (*InsertVersionDefinitionStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO version_definition (id, name) VALUES (?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for version_definition: %s", err)
	} else {
		return &InsertVersionDefinitionStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertVersionDefinitionStatement) Exec(data *VersionDefinition) (sql.Result, error) {
	result, err := s.stmt.Exec(data.Id, data.Name)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for version_definition %d: %s", data.Id, err)
	} else {
		return result, nil
	}
}

func (s *InsertVersionDefinitionStatement) Close() error {
	return s.stmt.Close()
}
