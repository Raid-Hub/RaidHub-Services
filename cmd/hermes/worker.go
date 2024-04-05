package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"
	"strconv"
	"sync"
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
		crawl_player_profiles(request.MembershipId, -1, db)
	} else if lastCrawled == nil || time.Since(*lastCrawled) > 24*time.Hour {
		log.Println("Crawling potentially stale player", request.MembershipId)
		crawl_membership(membershipType, request.MembershipId, db)
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

func crawl_player_profiles(destinyMembershipId string, membershipType int, db *sql.DB) {
	profiles, err := bungie.GetLinkedProfiles(membershipType, destinyMembershipId)
	if err != nil {
		log.Printf("Failed to get linked profiles: %s", err)
	} else if len(profiles) == 0 {
		log.Println("No profiles found")
		return
	}

	var wg sync.WaitGroup
	for _, profile := range profiles {
		wg.Add(1)
		go func(membershipId string) {
			defer wg.Done()
			crawl_membership(membershipType, membershipId, db)
		}(profile.MembershipId)
	}

	wg.Wait()
}

func crawl_membership(membershipType int, membershipId string, db *sql.DB) {
	profile, err := bungie.GetProfile(membershipType, membershipId)
	if err != nil {
		log.Printf("Failed to get profile: %s", err)
		return
	}

	if profile.Profile.Data == nil {
		log.Printf("Profile component is nil")
		return
	}

	if profile.Characters.Data == nil {
		log.Printf("Characters component is nil")
		return
	}

	tx, err := db.Begin()
	defer tx.Rollback()

	if err != nil {
		log.Printf("Failed to initiate transaction: %s", err)
		return
	}

	userInfo := profile.Profile.Data.UserInfo
	var bungieGlobalDisplayNameCodeStr *string = nil
	var bungieGlobalDisplayName *string = nil
	if userInfo.BungieGlobalDisplayName == nil || userInfo.BungieGlobalDisplayNameCode == nil || *userInfo.BungieGlobalDisplayName == "" {
		bungieGlobalDisplayName = nil
		bungieGlobalDisplayNameCodeStr = nil
	} else {
		bungieGlobalDisplayName = userInfo.BungieGlobalDisplayName
		bungieGlobalDisplayNameCodeStr = bungie.FixBungieGlobalDisplayNameCode(userInfo.BungieGlobalDisplayNameCode)
	}

	membershipIdInt64, err := strconv.ParseInt(userInfo.MembershipId, 10, 64)
	if err != nil {
		log.Printf("Failed to convert membershipId: %s", err)
		return
	}

	var mostRecentCharacter *bungie.DestinyCharacterComponent = nil
	var mostRecentDate *time.Time = nil
	for _, character := range *profile.Characters.Data {
		dateLastPlayed, err := time.Parse(time.RFC3339, character.DateLastPlayed)
		if err != nil {
			continue
		}

		if mostRecentCharacter == nil || mostRecentDate == nil || dateLastPlayed.After(*mostRecentDate) {
			mostRecentCharacter = &character
			mostRecentDate = &dateLastPlayed
		}
	}
	if mostRecentCharacter == nil {
		log.Println("No characters found")
		return
	}

	_, err = postgres.UpsertPlayer(tx, &postgres.Player{
		MembershipId:                membershipIdInt64,
		MembershipType:              &userInfo.MembershipType,
		IconPath:                    &mostRecentCharacter.EmblemPath,
		DisplayName:                 userInfo.DisplayName,
		BungieGlobalDisplayName:     bungieGlobalDisplayName,
		BungieGlobalDisplayNameCode: bungieGlobalDisplayNameCodeStr,
		LastSeen:                    *mostRecentDate,
	})
	if err != nil {
		log.Printf("Failed to upsert full player: %s", err)
		return
	}

	_, err = tx.Exec(`UPDATE player SET last_crawled = NOW() WHERE membership_id = $1`, membershipId)
	if err != nil {
		log.Printf("Failed to update last crawled: %s", err)
		return
	}
	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %s", err)
	} else {
		log.Printf("Upserted membership_id %s", membershipId)
	}
}
