package player_crawl

import (
	"database/sql"
	"encoding/json"
	"log"
	"raidhub/async"
	"raidhub/shared/bungie"
	"raidhub/shared/postgres"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func process_queue(qw *async.QueueWorker, msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		process_request(&msg, qw.Db)
	}
}

func process_request(msg *amqp.Delivery, db *sql.DB) {
	defer func() {
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}()

	var request PlayerRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Fatalf("Failed to unmarshal message: %s", err)
		return
	}

	membershipType, lastCrawled, err := get_player(request.MembershipId, db)
	if err != nil {
		log.Printf("Failed to get player: %s", err)
		return
	} else if membershipType == -1 || membershipType == 0 {
		log.Printf("Crawling missing player %d", request.MembershipId)
		crawl_player_profiles(request.MembershipId, db)
	} else if lastCrawled == nil || time.Since(*lastCrawled) > 24*time.Hour {
		log.Printf("Crawling potentially stale player %d/%d", membershipType, request.MembershipId)
		crawl_membership(membershipType, request.MembershipId, db)
	}
}

func get_player(membershipId int64, db *sql.DB) (int, *time.Time, error) {
	var membershipType int
	var lastCrawled sql.NullTime
	err := db.QueryRow(`SELECT COALESCE(membership_type, 0), last_crawled FROM player WHERE membership_id = $1 LIMIT 1`, membershipId).Scan(&membershipType, &lastCrawled)
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

func crawl_player_profiles(destinyMembershipId int64, db *sql.DB) {
	profiles, err := bungie.GetLinkedProfiles(-1, destinyMembershipId, true)
	if err != nil {
		log.Printf("Failed to get linked profiles: %s", err)
	} else if len(profiles) == 0 {
		log.Println("No profiles found")
		return
	}

	var wg sync.WaitGroup
	for _, profile := range profiles {
		wg.Add(1)
		go func(membershipId int64, membershipType int) {
			defer wg.Done()
			crawl_membership(membershipType, membershipId, db)
		}(profile.MembershipId, profile.MembershipType)
	}

	wg.Wait()
}

func crawl_membership(membershipType int, membershipId int64, db *sql.DB) {
	profile, err := bungie.GetProfile(membershipType, membershipId)
	if err != nil {
		log.Printf("Failed to get profile %d/%d: %s", membershipType, membershipId, err)
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

	var characterId int64
	for key := range *profile.Characters.Data {
		characterId = key
		break
	}

	_, activityHistoryErrorCode, activityHistoryErr := bungie.GetActivityHistoryPage(membershipType, membershipId, characterId, 0)
	if activityHistoryErr != nil && activityHistoryErrorCode != 1665 {
		log.Printf("Activity history error: %s", activityHistoryErr)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to initiate transaction: %s", err)
		return
	}
	defer tx.Rollback()

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

	var iconPath *string = nil
	var mostRecentDate time.Time = time.Time{}
	for _, character := range *profile.Characters.Data {
		if iconPath == nil || character.DateLastPlayed.After(mostRecentDate) {
			icon := character.EmblemPath
			iconPath = &icon
			mostRecentDate = character.DateLastPlayed
		}
	}

	_, err = postgres.UpsertPlayer(tx, &postgres.Player{
		MembershipId:                userInfo.MembershipId,
		MembershipType:              &userInfo.MembershipType,
		IconPath:                    iconPath,
		DisplayName:                 userInfo.DisplayName,
		BungieGlobalDisplayName:     bungieGlobalDisplayName,
		BungieGlobalDisplayNameCode: bungieGlobalDisplayNameCodeStr,
		LastSeen:                    mostRecentDate,
	})
	if err != nil {
		log.Printf("Failed to upsert full player: %s", err)
		return
	}

	_, err = tx.Exec(`UPDATE player SET last_crawled = NOW(), is_private = $1 WHERE membership_id = $2`, activityHistoryErrorCode == 1665, membershipId)
	if err != nil {
		log.Printf("Failed to update last crawled: %s", err)
		return
	}
	if err = tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %s", err)
	} else {
		log.Printf("Upserted membership_id %d", membershipId)
	}
}
