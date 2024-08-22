package pgcr

import (
	"context"
	"encoding/json"
	"log"
	"raidhub/async"
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
	batchSize = 8192
	batchTime = 60 * time.Minute
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
	var instances []clickhouse.ClickhouseInstance
	for _, msg := range msgs {
		var request ProcessedActivity
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			log.Println("Failed to unmarshal activity:", err)
			return
		}
		instances = append(instances, *parse(request))
	}

	err := clickhouse.InsertProcessedInstances(*c, instances)
	if err != nil {
		log.Fatalf("Failed to insert instances: %s", err)
	}

	success = true
}

func parse(request ProcessedActivity) *clickhouse.ClickhouseInstance {
	instance := clickhouse.ClickhouseInstance{
		"instance_id":    request.InstanceId,
		"hash":           request.Hash,
		"completed":      request.Completed,
		"player_count":   request.PlayerCount,
		"fresh":          2,
		"flawless":       2,
		"date_started":   request.DateStarted,
		"date_completed": request.DateCompleted,
		"platform_type":  uint16(request.MembershipType),
		"duration":       request.DurationSeconds,
		"score":          request.Score,
	}
	if request.Fresh != nil {
		if *request.Fresh {
			instance["fresh"] = 1
		} else {
			instance["fresh"] = 0
		}
	}
	if request.Flawless != nil {
		if *request.Flawless {
			instance["flawless"] = 1
		} else {
			instance["flawless"] = 0
		}
	}

	players := make([]map[string]interface{}, len(request.Players))

	for i, player := range request.Players {
		instancePlayer := map[string]interface{}{
			"membership_id":       player.Player.MembershipId,
			"completed":           player.Finished,
			"time_played_seconds": player.TimePlayedSeconds,
			"sherpas":             player.Sherpas,
			"is_first_clear":      player.IsFirstClear,
		}
		characters := make([]map[string]interface{}, len(player.Characters))

		for j, character := range player.Characters {
			instanceCharacter := map[string]interface{}{
				"character_id":        character.CharacterId,
				"class_hash":          0,
				"emblem_hash":         0,
				"completed":           character.Completed,
				"score":               character.Score,
				"kills":               character.Kills,
				"assists":             character.Assists,
				"deaths":              character.Deaths,
				"precision_kills":     character.PrecisionKills,
				"super_kills":         character.SuperKills,
				"grenade_kills":       character.GrenadeKills,
				"melee_kills":         character.MeleeKills,
				"time_played_seconds": character.TimePlayedSeconds,
				"start_seconds":       character.StartSeconds,
			}
			if character.ClassHash != nil {
				instanceCharacter["class_hash"] = *character.ClassHash
			}
			if character.EmblemHash != nil {
				instanceCharacter["emblem_hash"] = *character.EmblemHash
			}
			weapons := make([]map[string]interface{}, len(character.Weapons))

			for k, weapon := range character.Weapons {
				instanceCharacterWeapon := map[string]interface{}{
					"weapon_hash":     weapon.WeaponHash,
					"kills":           weapon.Kills,
					"precision_kills": weapon.PrecisionKills,
				}
				weapons[k] = instanceCharacterWeapon
			}
			instanceCharacter["weapons"] = weapons
			characters[j] = instanceCharacter
		}
		instancePlayer["characters"] = characters
		players[i] = instancePlayer
	}
	instance["players"] = players
	return &instance
}
