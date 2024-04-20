package clickhouse

import (
	"database/sql"
	"fmt"
	"time"
)

type Player struct {
	MembershipId                                                                            int64
	MembershipType                                                                          uint16
	IconPath, DisplayName, BungieGlobalDisplayName, BungieGlobalDisplayNameCode, BungieName string
	LastSeen                                                                                time.Time
	Clears, FreshClears, Sherpas                                                            int
	SumOfBest                                                                               int32
}

type InsertPlayerStatement struct {
	stmt *sql.Stmt
}

func (c *ClickhouseClient) PreparePlayer(tx *sql.Tx) (*InsertPlayerStatement, error) {
	stmt, err := tx.Prepare(`INSERT INTO player (membership_id, membership_type, icon_path, display_name, bungie_global_display_name, bungie_global_display_name_code, bungie_name, last_seen, clears, fresh_clears, sherpas, sum_of_best) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return nil, fmt.Errorf("error preparing statement for player: %s", err)
	} else {
		return &InsertPlayerStatement{
			stmt: stmt,
		}, nil
	}
}

func (s *InsertPlayerStatement) Exec(data *Player) (sql.Result, error) {
	result, err := s.stmt.Exec(data.MembershipId, data.MembershipType, data.IconPath, data.DisplayName, data.BungieGlobalDisplayName, data.BungieGlobalDisplayNameCode, data.BungieName, data.LastSeen, data.Clears, data.FreshClears, data.Sherpas, data.SumOfBest)
	if err != nil {
		return nil, fmt.Errorf("error executing statement for player %d: %s", data.MembershipId, err)
	} else {
		return result, nil
	}
}

func (s *InsertPlayerStatement) Close() error {
	return s.stmt.Close()
}
