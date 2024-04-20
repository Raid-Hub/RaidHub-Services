package clickhouse

import (
	"database/sql"
	"fmt"
)

type InstanceCharacterWeapon struct {
	InstanceId     int64
	MembershipId   int64
	CharacterId    int64
	WeaponHash     int64
	Kills          int
	PrecisionKills int
}

type InsertInstanceCharacterWeaponStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareInstanceCharacterWeapon(tx *sql.Tx) (*InsertInstanceCharacterWeaponStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO instance_character_weapon (instance_id, membership_id, character_id, weapon_hash, kills, precision_kills) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for instance_character_weapon: %s", err)
	} else {
		return &InsertInstanceCharacterWeaponStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertInstanceCharacterWeaponStatement) Exec(data *InstanceCharacterWeapon) (sql.Result, error) {
	result, err := s.stmt.Exec(data.InstanceId, data.MembershipId, data.CharacterId, data.WeaponHash, data.Kills, data.PrecisionKills)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for instance_character_weapon %d: %s", data.InstanceId, err)
	} else {
		return result, nil
	}
}

func (s *InsertInstanceCharacterWeaponStatement) Close() error {
	return s.stmt.Close()
}
