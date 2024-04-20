package pgcr

import (
	"context"
	"encoding/json"
	"log"
	"raidhub/shared/async"
	"raidhub/shared/clickhouse"

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

func process_queue(qw *async.QueueWorker, msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		process_request(&msg, qw.Clickhouse)
	}
}

func process_request(msg *amqp.Delivery, c *clickhouse.ClickhouseClient) {
	defer func() {
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}()

	var request ProcessedActivity
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Failed to unmarshal message: %s", err)
		return
	}

	log.Println("Clickhouse", request.InstanceId)
}

// // Clickhouse
// ctx, err := client.Begin()
// if err != nil {
// 	log.Fatal("Error beginning ClickHouse transaction:", err)
// }
// defer ctx.Rollback()

// stmnt, err := client.PrepareInstance(ctx)
// if err != nil {
// 	log.Fatal("Error preparing statement for instance:", err)
// }
// defer stmnt.Close()

// instance := clickhouse.Instance{
// 	InstanceId:    pgcr.InstanceId,
// 	Hash:          pgcr.Hash,
// 	Completed:     pgcr.Completed,
// 	PlayerCount:   pgcr.PlayerCount,
// 	DateStarted:   pgcr.DateStarted,
// 	DateCompleted: pgcr.DateCompleted,
// 	PlatformType:  uint16(pgcr.MembershipType),
// 	Duration:      pgcr.DurationSeconds,
// 	Score:         pgcr.Score,
// }
// if pgcr.Fresh != nil {
// 	if *pgcr.Fresh {
// 		instance.Fresh = 1
// 	} else {
// 		instance.Fresh = 0
// 	}
// } else {
// 	instance.Fresh = 2
// }

// if pgcr.Flawless != nil {
// 	if *pgcr.Flawless {
// 		instance.Flawless = 1
// 	} else {
// 		instance.Flawless = 0
// 	}
// } else {
// 	instance.Flawless = 2
// }

// _, err = stmnt.Exec(&instance)
// if err != nil {
// 	log.Fatal("Error inserting instance:", err)
// }

// stmntInstancePlayer, err := client.PrepareInstancePlayer(ctx)
// if err != nil {
// 	log.Fatal("Error preparing statement for instance player:", err)
// }
// defer stmntInstancePlayer.Close()

// for _, playerActivity := range pgcr.PlayerActivities {
// 	instancePlayer := clickhouse.InstancePlayer{
// 		InstanceId:        pgcr.InstanceId,
// 		MembershipId:      playerActivity.Player.MembershipId,
// 		Completed:         playerActivity.Finished,
// 		TimePlayedSeconds: playerActivity.TimePlayedSeconds,
// 		IsFirstClear:      playerActivity,
// 		Sherpas:           playerActivity.Sherpas,
// 	}
// 	_, err = stmntInstancePlayer.Exec(&instancePlayer)
// 	if err != nil {
// 		log.Fatal("Error inserting instance player:", err)
// 	}
// }

// err = ctx.Commit()
// if err != nil {
// 	log.Fatal(err)
// 	return nil, false, err
// }
