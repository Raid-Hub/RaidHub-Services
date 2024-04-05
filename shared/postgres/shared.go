package postgres

import (
	"database/sql"
	"time"
)

type Player struct {
	MembershipId                int64
	MembershipType              *int
	LastSeen                    time.Time
	IconPath                    *string
	DisplayName                 *string
	BungieGlobalDisplayName     *string
	BungieGlobalDisplayNameCode *string
	Full                        bool
}

func UpsertFullPlayer(tx *sql.Tx, player *Player) error {
	_, err := tx.Exec(`
			INSERT INTO player (
				"membership_id",
				"membership_type",
				"icon_path",
				"display_name",
				"bungie_global_display_name",
				"bungie_global_display_name_code",
				"last_seen"
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7
			)
			ON CONFLICT (membership_id)
			DO UPDATE SET
				membership_type = COALESCE(player.membership_type, EXCLUDED.membership_type),
				icon_path = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN COALESCE(EXCLUDED.icon_path, player.icon_path)
					ELSE player.icon_path
				END,
				display_name = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.display_name
					ELSE player.display_name
				END,
				bungie_global_display_name = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.bungie_global_display_name
					ELSE player.bungie_global_display_name
				END,
				bungie_global_display_name_code = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.bungie_global_display_name_code
					ELSE player.bungie_global_display_name_code
				END,
				last_seen = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.last_seen
					ELSE player.last_seen
				END;
			`,
		player.MembershipId, player.MembershipType, player.IconPath, player.DisplayName,
		player.BungieGlobalDisplayName, player.BungieGlobalDisplayNameCode, player.LastSeen)
	return err
}
