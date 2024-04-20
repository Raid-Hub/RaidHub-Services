package character_fill

import (
	"context"
	"encoding/json"
	"raidhub/shared/async"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type CharacterFillRequest struct {
	MembershipId string `json:"membershipId"`
	CharacterId  string `json:"characterId"`
	InstanceId   string `json:"instanceId"`
}

const queueName = "character_fill"

func Create() async.QueueWorker {
	return async.QueueWorker{
		QueueName: queueName,
		Processer: process_queue,
	}
}

func SendMessage(ch *amqp.Channel, membershipId int64, characterId int64, instanceId int64) error {
	body, err := json.Marshal(CharacterFillRequest{
		MembershipId: strconv.FormatInt(membershipId, 10),
		CharacterId:  strconv.FormatInt(characterId, 10),
		InstanceId:   strconv.FormatInt(instanceId, 10),
	})
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
