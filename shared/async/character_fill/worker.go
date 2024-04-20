package character_fill

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/bungie"

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

	var request CharacterFillRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Failed to unmarshal message: %s", err)
		return
	}

	var membershipTypeValue sql.NullInt32
	db.QueryRow("SELECT membership_type FROM player WHERE membership_id = $1", request.MembershipId).Scan(&membershipTypeValue)

	var membershipType int32
	if membershipTypeValue.Valid && membershipTypeValue.Int32 > 0 {
		log.Printf("character membership type identified: %d", membershipTypeValue.Int32)
		membershipType = membershipTypeValue.Int32
	} else {
		log.Println("character membership type not found")
		profiles, err := bungie.GetLinkedProfiles(-1, request.MembershipId, false)
		if err != nil {
			log.Printf("Failed to get linked profile: %s", err)
			return
		}
		for _, p := range profiles {
			if p.MembershipId == request.MembershipId {
				membershipType = int32(p.MembershipType)
				break
			}
		}
	}
	char, err := bungie.GetCharacter(membershipType, request.MembershipId, request.CharacterId)
	if err != nil {
		log.Printf("Failed to get character: %s", err)
		return
	}

	if char.Character == nil {
		log.Printf("Failed to get character: %s", fmt.Errorf("no character component in the response"))
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE activity_character
		SET class_hash = $1
		WHERE membership_id = $2 AND instance_id = $3 AND character_id = $4`,
		char.Character.Data.ClassHash, request.MembershipId, request.InstanceId, request.CharacterId)
	if err != nil {
		log.Fatal(err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if count != 1 {
		log.Printf("Failed to update activity_character, rows found: %d", count)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Updated character,activity: %s,%s", request.CharacterId, request.InstanceId)
}
