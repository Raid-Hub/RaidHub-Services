package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PlayerRequest struct {
	MembershipId string `json:"membershipId"`
}

func process_queue(msgs <-chan amqp.Delivery, db *sql.DB) {
	for msg := range msgs {
		var request PlayerRequest
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			log.Printf("Failed to unmarshal message: %s", err)
			continue
		}
		process_request(&request, db)
	}
}

func process_request(request *PlayerRequest, db *sql.DB) {
	membershipType, lastCrawled, err := get_player(request.MembershipId, db)
	if err != nil {
		log.Printf("Failed to get player: %s", err)
		return
	} else if membershipType == -1 {
		log.Println("Crawling missing player", request.MembershipId)
		crawl_player_profile(request.MembershipId, -1, db)
	} else if lastCrawled == nil || time.Since(*lastCrawled) > 24*time.Hour {
		log.Println("Crawling potentially stale player", request.MembershipId)
		crawl_player_profile(request.MembershipId, membershipType, db)
	}
}

func get_player(membershipId string, db *sql.DB) (int, *time.Time, error) {
	var membershipType int
	var lastCrawled sql.NullTime
	err := db.QueryRow(`SELECT membership_type, last_crawled FROM player WHERE membership_id = $1 LIMIT 1`, membershipId).Scan(&membershipType, &lastCrawled)
	if err == sql.ErrNoRows {
		return -1, nil, nil
	} else if err != nil {
		return -1, nil, err
	} else {
		if lastCrawled.Valid {
			return membershipType, &lastCrawled.Time, nil
		} else {
			return membershipType, nil, nil
		}
	}
}

func crawl_player_profile(destinyMembershipId string, membershipType int, db *sql.DB) {
	profiles, err := bungie.GetLinkedProfiles(membershipType, destinyMembershipId)
	if err != nil {
		log.Printf("Failed to get linked profiles: %s", err)
	} else if len(profiles) == 0 {
		log.Println("No profiles found")
		return
	}
	for _, profile := range profiles {
		tx, err := db.Begin()
		defer tx.Rollback()

		if err != nil {
			log.Printf("Failed to initiate transaction: %s", err)
			continue
		}

		var bungieGlobalDisplayNameCodeStr *string = nil
		if profile.BungieGlobalDisplayName == nil || profile.BungieGlobalDisplayNameCode == nil || *profile.BungieGlobalDisplayName == "" {
			profile.BungieGlobalDisplayName = nil
			bungieGlobalDisplayNameCodeStr = nil
		} else {
			bungieGlobalDisplayNameCodeStr = bungie.FixBungieGlobalDisplayNameCode(profile.BungieGlobalDisplayNameCode)
		}

		lastSeen, err := time.Parse(time.RFC3339, profile.DateLastPlayed)
		if err != nil {
			log.Printf("Failed to parse last seen: %s", err)
			continue
		}

		membershipId, err := strconv.ParseInt(profile.MembershipId, 10, 64)
		if err != nil {
			log.Printf("Failed to convert membershipId: %s", err)
			continue
		}

		err = postgres.UpsertFullPlayer(tx, &postgres.Player{
			MembershipId:                membershipId,
			MembershipType:              &profile.MembershipType,
			IconPath:                    profile.IconPath,
			DisplayName:                 profile.DisplayName,
			BungieGlobalDisplayName:     profile.BungieGlobalDisplayName,
			BungieGlobalDisplayNameCode: bungieGlobalDisplayNameCodeStr,
			LastSeen:                    lastSeen,
			Full:                        true,
		})
		if err != nil {
			log.Printf("Failed to upsert full player: %s", err)
			break
		}

		_, err = tx.Exec(`UPDATE player SET last_crawled = NOW() WHERE membership_id = $1`, membershipId)
		if err != nil {
			log.Printf("Failed to update last crawled: %s", err)
			break
		}
		if err = tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction: %s", err)
		} else {
			log.Printf("Upserted membership_id %s", profile.MembershipId)
		}
	}
}
