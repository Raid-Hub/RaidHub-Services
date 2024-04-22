package pgcr

import (
	"context"
	"encoding/json"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/clickhouse"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	amqp "github.com/rabbitmq/amqp091-go"
)

const queueName = "pgcr_clickhouse"

func CreateClickhouseQueue() async.QueueWorker {
	return async.QueueWorker{
		QueueName: queueName,
		Processer: process_queue,
	}
}

func SendToClickhouse(ch *amqp.Channel, activity *ProcessedActivity) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

const (
	batchSize = 1024
	batchTime = 10 * time.Minute
)

func process_queue(qw *async.QueueWorker, msgs <-chan amqp.Delivery) {

	batch := make([]amqp.Delivery, 0, batchSize)
	timer := time.NewTimer(batchTime)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				// Channel is closed, process remaining messages and return
				if len(batch) > 0 {
					process(batch, qw.Clickhouse)
				}
				return
			}

			batch = append(batch, msg)

			if len(batch) >= batchSize {
				process(batch, qw.Clickhouse)
				batch = batch[:0]
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(batchTime)
			}
		case <-timer.C:
			if len(batch) > 0 {
				process(batch, qw.Clickhouse)
				batch = batch[:0]
			}
			timer.Reset(batchTime)
		}
	}
}

func process(msgs []amqp.Delivery, c *driver.Conn) {
	var success bool
	defer func() {
		if !success {
			log.Printf("failed to send %d instances to clickhouse, rejecting all", len(msgs))
		} else {
			log.Printf("sent %d instances to clickhouse", len(msgs))
		}
		for _, msg := range msgs {
			if success {
				if err := msg.Ack(false); err != nil {
					log.Printf("Failed to acknowledge messages: %v", err)
				}
			} else {
				if err := msg.Reject(true); err != nil {
					log.Printf("Failed to acknowledge messages: %v", err)
				}
			}
		}

	}()

	var instances []clickhouse.Instance
	var players []clickhouse.InstancePlayer
	var characters []clickhouse.InstanceCharacter
	var weapons []clickhouse.InstanceCharacterWeapon

	for _, msg := range msgs {
		var request ProcessedActivity
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			log.Println("Failed to unmarshal message:", err)
			return
		}

		instance, newPlayers, newCharacters, newWeapons := parse(request)
		instances = append(instances, *instance)
		players = append(players, newPlayers...)
		characters = append(characters, newCharacters...)
		weapons = append(weapons, newWeapons...)
	}

	err := clickhouse.InsertInstances(*c, instances)
	if err != nil {
		log.Fatalf("Failed to insert instances: %s", err)
	}

	err = clickhouse.InsertInstancePlayers(*c, players)
	if err != nil {
		log.Fatalf("Failed to insert players: %s", err)
	}

	err = clickhouse.InsertInstanceCharacters(*c, characters)
	if err != nil {
		log.Fatalf("Failed to insert characters: %s", err)
	}

	err = clickhouse.InsertCharacterWeapons(*c, weapons)
	if err != nil {
		log.Fatalf("Failed to insert weapons: %s", err)
	}

	success = true
}

func parse(request ProcessedActivity) (*clickhouse.Instance, []clickhouse.InstancePlayer, []clickhouse.InstanceCharacter, []clickhouse.InstanceCharacterWeapon) {
	instance := clickhouse.Instance{
		InstanceId:    request.InstanceId,
		Hash:          request.Hash,
		Completed:     request.Completed,
		PlayerCount:   request.PlayerCount,
		DateStarted:   request.DateStarted,
		DateCompleted: request.DateCompleted,
		PlatformType:  uint16(request.MembershipType),
		Duration:      request.DurationSeconds,
		Score:         request.Score,
	}
	if request.Fresh == nil {
		instance.Fresh = 2
	} else if *request.Fresh {
		instance.Fresh = 1
	} else {
		instance.Fresh = 0
	}

	if request.Flawless == nil {
		instance.Flawless = 2
	} else if *request.Flawless {
		instance.Flawless = 1
	} else {
		instance.Flawless = 0
	}

	var players []clickhouse.InstancePlayer
	var characters []clickhouse.InstanceCharacter
	var weapons []clickhouse.InstanceCharacterWeapon

	for _, player := range request.Players {
		instancePlayer := clickhouse.InstancePlayer{
			InstanceId:        request.InstanceId,
			MembershipId:      player.Player.MembershipId,
			Completed:         player.Finished,
			TimePlayedSeconds: player.TimePlayedSeconds,
			Sherpas:           player.Sherpas,
			IsFirstClear:      player.IsFirstClear,
		}
		players = append(players, instancePlayer)

		for _, character := range player.Characters {
			instanceCharacter := clickhouse.InstanceCharacter{
				InstanceId:        request.InstanceId,
				MembershipId:      player.Player.MembershipId,
				CharacterId:       character.CharacterId,
				Completed:         character.Completed,
				Score:             character.Score,
				Kills:             character.Kills,
				Assists:           character.Assists,
				Deaths:            character.Deaths,
				PrecisionKills:    character.PrecisionKills,
				SuperKills:        character.SuperKills,
				GrenadeKills:      character.GrenadeKills,
				MeleeKills:        character.MeleeKills,
				TimePlayedSeconds: character.TimePlayedSeconds,
				StartSeconds:      character.StartSeconds,
			}
			if character.ClassHash != nil {
				instanceCharacter.ClassHash = *character.ClassHash
			}
			if character.EmblemHash != nil {
				instanceCharacter.EmblemHash = *character.EmblemHash
			}
			characters = append(characters, instanceCharacter)

			for _, weapon := range character.Weapons {
				instanceCharacterWeapon := clickhouse.InstanceCharacterWeapon{
					InstanceId:     request.InstanceId,
					MembershipId:   player.Player.MembershipId,
					CharacterId:    character.CharacterId,
					WeaponHash:     weapon.WeaponHash,
					Kills:          weapon.Kills,
					PrecisionKills: weapon.PrecisionKills,
				}
				weapons = append(weapons, instanceCharacterWeapon)
			}
		}
	}
	return &instance, players, characters, weapons
}
