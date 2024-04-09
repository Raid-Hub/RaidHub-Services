package bonus_pgcr

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"raidhub/shared/async"
	"raidhub/shared/discord"
	"raidhub/shared/pgcr"
	"strconv"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	outgoing *amqp.Channel
	once     sync.Once
)

func create_outbound_channel() {
	once.Do(func() {
		conn, err := async.Init()
		if err != nil {
			log.Fatalf("Failed to create outbound channel: %s", err)
		}
		outgoing, _ = conn.Channel()
	})
}

func process_queue(msgs <-chan amqp.Delivery, db *sql.DB) {
	client := &http.Client{}
	apiKey := os.Getenv("BUNGIE_API_KEY")
	baseUrl := os.Getenv("PGCR_URL_BASE")

	create_outbound_channel()
	for msg := range msgs {
		process_request(&msg, db, client, baseUrl, apiKey)
	}
}

func process_request(msg *amqp.Delivery, db *sql.DB, client *http.Client, baseUrl string, apiKey string) {
	defer func() {
		if err := msg.Ack(false); err != nil {
			log.Printf("Failed to acknowledge message: %v", err)
		}
	}()

	var request PGCRRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Failed to unmarshal message: %s", err)
		return
	}

	log.Printf("Checking bonus pgcr %s", request.InstanceId)
	exists, err := check_if_pgcr_exists(request.InstanceId, db)
	if err != nil {
		log.Printf("Error reading database for pgcr request %s: %s", request.InstanceId, err)
	} else if exists {
		log.Printf("%s already exists", request.InstanceId)
		return
	} else {
		instanceIdInt, err := strconv.ParseInt(request.InstanceId, 10, 64)
		if err != nil {
			log.Printf("Error parsing instance_id %s: %s", request.InstanceId, err)
			return
		}
		result, _ := pgcr.FetchAndStorePGCR(client, instanceIdInt, db, outgoing, baseUrl, apiKey)
		if result == pgcr.Success {
			// todo: track new pgcrs
			log.Printf("%s added to data set", request.InstanceId)

			msg := fmt.Sprintf("Found missing PGCR: %d", instanceIdInt)
			webhook := discord.Webhook{
				Content: &msg,
			}
			discord.SendWebhook(os.Getenv("PAN_WEBHOOK_URL"), &webhook)
		} else if result == pgcr.AlreadyExists {
			log.Printf("%s is already added: %d", request.InstanceId, result)
		} else {
			log.Printf("%s returned an error result: %d", request.InstanceId, result)
			pgcr.WriteMissedLog(instanceIdInt)
		}
	}
}

func check_if_pgcr_exists(instanceid string, db *sql.DB) (bool, error) {
	var result bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM activity a INNER JOIN pgcr ON a.instance_id = pgcr.instance_id WHERE a.instance_id = $1 LIMIT 1)`, instanceid).Scan(&result)
	if err != nil {
		return false, err
	} else {
		return result, nil
	}
}
