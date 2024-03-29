package pgcr

import (
	"database/sql"
	"log"
	"time"

	"github.com/lib/pq"
)

// Returns lag, is_new, err
func StorePGCR(pgcr *ProcessedActivity, raw *DestinyPostGameCarnageReport, postgresDb *sql.DB) (*time.Duration, bool, error) {
	// Identify the raid which this PGCR belongs to
	var raidId int
	err := postgresDb.QueryRow(`SELECT raid_id FROM raid_definition WHERE hash = $1`, pgcr.RaidHash).Scan(&raidId)
	if err != nil {
		log.Printf("Error finding raid_id for %d", pgcr.RaidHash)
		return nil, false, err
	}

	lag := time.Since(pgcr.DateCompleted)

	// Store the raw JSON
	err = StoreJSON(raw, postgresDb)
	if err != nil {
		log.Println("Failed to store raw JSON")
		return nil, false, err
	}

	tx, err := postgresDb.Begin()
	if err != nil {
		log.Println("Failed to initiate transaction")
		return nil, false, err
	}

	defer tx.Rollback()

	// Nothing should happen if this fails
	_, err = tx.Exec(`INSERT INTO "activity" (
		"instance_id",
		"raid_hash",
		"flawless",
		"completed",
		"fresh",
		"player_count",
		"date_started",
		"date_completed",
		"platform_type",
		"duration"
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, pgcr.InstanceId, pgcr.RaidHash,
		pgcr.Flawless, pgcr.Completed, pgcr.Fresh, pgcr.PlayerCount,
		pgcr.DateStarted, pgcr.DateCompleted, pgcr.MembershipType, pgcr.DurationSeconds)

	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && (pqErr.Code == "23503" || pqErr.Code == "23505") {
			// Handle the duplicate key error here
			log.Printf("Duplicate instanceId: %d", pgcr.InstanceId)
			return &lag, false, nil
		} else {
			log.Printf("Error inserting activity into DB for instanceId %d", pgcr.InstanceId)
			return nil, false, err
		}
	}

	completedDictionary := map[int64]bool{}
	fastestClearSoFar := map[int64]int{}
	for _, playerActivity := range pgcr.PlayerActivities {
		var playerRaidClearCount int
		var duration int
		// the sum is a null hack, but it finds distinct rows anyways
		err = tx.QueryRow(`
			SELECT COALESCE(SUM(ps.clears), 0) AS count, COALESCE(SUM(a.duration), 100000000)
			FROM player_stats ps
			LEFT JOIN activity a ON ps.fastest_instance_id = a.instance_id
			WHERE ps.membership_id = $1 AND ps.raid_id = $2`,
			playerActivity.Player.MembershipId, raidId).
			Scan(&playerRaidClearCount, &duration)
		fastestClearSoFar[playerActivity.Player.MembershipId] = duration

		if err != nil {
			log.Printf("Error querying clears in DB for instance_id, membership_id, raid_id: %d, %d, %d", pgcr.InstanceId, playerActivity.Player.MembershipId, raidId)
			return nil, false, err
		}

		if playerActivity.DidFinish {
			completedDictionary[playerActivity.Player.MembershipId] = playerRaidClearCount > 0
		}

		// handle various player response types, full and partial
		if playerActivity.Player.Full {
			_, err = tx.Exec(`
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
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.icon_path
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
			`, playerActivity.Player.MembershipId, playerActivity.Player.MembershipType,
				playerActivity.Player.IconPath, playerActivity.Player.DisplayName, playerActivity.Player.BungieGlobalDisplayName,
				playerActivity.Player.BungieGlobalDisplayNameCode, playerActivity.Player.LastSeen)

			if err != nil {
				log.Printf("Error inserting player (full) %d into DB for instanceId %d: %s",
					playerActivity.Player.MembershipId, pgcr.InstanceId, err)
				return nil, false, err
			}
		} else {
			// handle the partial response
			_, err := tx.Exec(`
			INSERT INTO player (
				"membership_id",
				"last_seen"
			)
			VALUES (
				$1, $2
			)
			ON CONFLICT (membership_id)
			DO UPDATE SET
				last_seen = CASE 
					WHEN EXCLUDED.last_seen > player.last_seen THEN EXCLUDED.last_seen
					ELSE player.last_seen
				END;
			`, playerActivity.Player.MembershipId, playerActivity.Player.LastSeen)

			if err != nil {
				log.Printf("Error inserting player (partial) %d into DB for instanceId %d: %s",
					playerActivity.Player.MembershipId, pgcr.InstanceId, err)
				return nil, false, err
			}
		}

		_, err = tx.Exec(`
			INSERT INTO "player_activity" (
				"instance_id",
				"membership_id",
				"finished_raid",
				"kills",
				"assists",
				"deaths",
				"time_played_seconds",
				"class_hash"
			) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`,
			pgcr.InstanceId, playerActivity.Player.MembershipId,
			playerActivity.DidFinish, playerActivity.Kills, playerActivity.Assists, playerActivity.Deaths,
			playerActivity.TimePlayedSeconds, playerActivity.ClassHash)
		if err != nil {
			log.Printf("Error inserting player_activity into DB for instanceId, membershipId %d, %d: %s", pgcr.InstanceId,
				playerActivity.Player.MembershipId, err)
			return nil, false, err
		}

		// update the player_stats table
		_, err = tx.Exec(`INSERT INTO player_stats ("membership_id", "raid_id")
			VALUES ($1, $2)
			ON CONFLICT (membership_id, raid_id)
			DO NOTHING`, playerActivity.Player.MembershipId, raidId)

		if err != nil {
			log.Printf("Error inserting player_stats into DB for membershipId, raid_id: %d, %d", playerActivity.Player.MembershipId, raidId)
			return nil, false, err
		}
	}

	// determine if a sherpa took place
	noobs := 0
	anyPro := false
	for _, hasClears := range completedDictionary {
		if hasClears {
			anyPro = true
		} else {
			noobs++
		}
	}

	sherpasHappened := anyPro && noobs > 0
	if sherpasHappened {
		log.Printf("Found %d sherpas for instance %d", noobs, pgcr.InstanceId)
	}

	for membershipId, hasClears := range completedDictionary {
		sherpaCount := 0
		if hasClears && sherpasHappened {
			sherpaCount = noobs

			// set sherpas for p_activity
			_, err = tx.Exec(`UPDATE 
				player_activity
			SET 
				sherpas = $1
			WHERE 
				membership_id = $2 AND
				instance_id = $3`, sherpaCount, membershipId, pgcr.InstanceId)

			if err != nil {
				log.Printf("Error updating sherpa count for player_activity with instanceId, membershipId %d, %d", pgcr.InstanceId, membershipId)
				return nil, false, err
			}

		} else if !hasClears {
			// first clear, update p_activity
			_, err = tx.Exec(`UPDATE 
				player_activity
			SET 
				is_first_clear = true
			WHERE 
				membership_id = $1 AND
				instance_id = $2`, membershipId, pgcr.InstanceId)

			if err != nil {
				log.Printf("Error updating first clear for instanceId, membershipId %d, %d", pgcr.InstanceId, membershipId)
				return nil, false, err
			}
		}

		// raid specific stats
		_, err = tx.Exec(`UPDATE player_stats 
			SET 
				sherpas = player_stats.sherpas + $3,
				clears = player_stats.clears + 1,
				fresh_clears = CASE
						WHEN $4 = true THEN player_stats.fresh_clears + 1
						ELSE player_stats.fresh_clears
					END,
				trios = CASE 
						WHEN $5 = 3 THEN player_stats.trios + 1
						ELSE player_stats.trios
					END,
				duos = CASE 
						WHEN $5 = 2 THEN player_stats.duos + 1
						ELSE player_stats.duos
					END,
				solos = CASE 
						WHEN $5 = 1 THEN player_stats.solos + 1
						ELSE player_stats.solos
					END,
				fastest_instance_id = CASE
						WHEN $4 = true AND $6::int < $7::int THEN $8::bigint
						ELSE player_stats.fastest_instance_id
					END
			WHERE
				membership_id = $1 AND
				raid_id = $2;
			`, membershipId, raidId, sherpaCount, pgcr.Fresh, pgcr.PlayerCount, pgcr.DurationSeconds, fastestClearSoFar[membershipId], pgcr.InstanceId)

		if err != nil {
			log.Printf("Error updating player_stats for membershipId %d", membershipId)
			return nil, false, err
		}

		// global stats
		_, err = tx.Exec(`UPDATE player 
			SET 
				clears = player.clears + 1,
				sherpas = player.sherpas + $2,
				fresh_clears = CASE 
						WHEN $3 = true THEN player.fresh_clears + 1
						ELSE player.fresh_clears
					END
			WHERE membership_id = $1`, membershipId, sherpaCount, pgcr.Fresh)

		if err != nil {
			log.Printf("Error updating global stats for membershipId %d", membershipId)
			return nil, false, err
		}

		if *pgcr.Fresh && pgcr.DurationSeconds < fastestClearSoFar[membershipId] {
			_, err = tx.Exec(`WITH c AS (SELECT COUNT(*) as expected FROM raid WHERE is_sunset = false)
				UPDATE player p
				SET sum_of_best = ptd.total_duration
				FROM (
					SELECT
						ps.membership_id,
						SUM(a.duration) AS total_duration
					FROM player_stats ps
					JOIN raid r ON ps.raid_id = r.id
					LEFT JOIN activity a ON ps.fastest_instance_id = a.instance_id
					WHERE a.duration IS NOT NULL AND is_sunset = false 
						AND ps.membership_id = $1
					GROUP BY ps.membership_id
					HAVING COUNT(a.instance_id) = (SELECT expected FROM c)
				) ptd
				WHERE p.membership_id = ptd.membership_id;`, membershipId)

			if err != nil {
				log.Printf("Error updating sum of best for membershipId %d", membershipId)
				return nil, false, err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
		return nil, false, err
	}
	return &lag, true, nil
}
