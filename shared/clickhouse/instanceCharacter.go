package clickhouse

import (
	"database/sql"
	"fmt"
)

type InstanceCharacter struct {
	InstanceId        int64
	MembershipId      int64
	CharacterId       int64
	ClassHash         uint32
	EmblemHash        uint32
	Completed         bool
	Score             int
	Kills             int
	Assists           int
	Deaths            int
	PrecisionKills    int
	SuperKills        int
	GrenadeKills      int
	MeleeKills        int
	TimePlayedSeconds int
	StartSeconds      int
}

type InsertInstanceCharacterStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PrepareInstanceCharacter(tx *sql.Tx) (*InsertInstanceCharacterStatement, error) {
	stmt, err := tx.Prepare("INSERT INTO instance_character (instance_id, membership_id, character_id, class_hash, emblem_hash, completed, score, kills, assists, deaths, precision_kills, super_kills, grenade_kills, melee_kills, time_played_seconds, start_seconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for instance_character: %s", err)
	} else {
		return &InsertInstanceCharacterStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertInstanceCharacterStatement) Exec(data *InstanceCharacter) (sql.Result, error) {
	result, err := s.stmt.Exec(data.InstanceId, data.MembershipId, data.CharacterId, data.ClassHash, data.EmblemHash, data.Completed, data.Score, data.Kills, data.Assists, data.Deaths, data.PrecisionKills, data.SuperKills, data.GrenadeKills, data.MeleeKills, data.TimePlayedSeconds, data.StartSeconds)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for instance_character %d: %s", data.InstanceId, err)
	} else {
		return result, nil
	}
}

func (s *InsertInstanceCharacterStatement) Close() error {
	return s.stmt.Close()
}
