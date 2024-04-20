package clickhouse

import (
	"database/sql"
	"fmt"
)

type WeaponDefinition struct {
	Hash     uint32
	Name     string
	IconPath string
}

type InsertWeaponDefinitionStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareWeaponDefinition(tx *sql.Tx) (*InsertWeaponDefinitionStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO weapon_definition (hash, name, icon_path) VALUES (?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for weapon_definition: %s", err)
	} else {
		return &InsertWeaponDefinitionStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertWeaponDefinitionStatement) Exec(data *WeaponDefinition) (sql.Result, error) {
	result, err := s.stmt.Exec(data.Hash, data.Name, data.IconPath)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for weapon_definition %d: %s", data.Hash, err)
	} else {
		return result, nil
	}
}

func (s *InsertWeaponDefinitionStatement) Close() error {
	return s.stmt.Close()
}
