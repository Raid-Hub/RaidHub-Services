package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"raidhub/shared/pgcr"

	amqp "github.com/rabbitmq/amqp091-go"
)

func malformedWorker(ch chan int64, rabbitChannel *amqp.Channel, db *sql.DB) {
	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	for _instanceId := range ch {
		// Spawn a worker for each instanceId
		go func(instanceId int64) {
			startTime := time.Now()
			for i := 1; i <= 5; i++ {
				result, lag := pgcr.FetchAndStorePGCR(client, instanceId, db, rabbitChannel, proxy, securityKey)

				if result == pgcr.AlreadyExists || result == pgcr.NonRaid {
					break
				} else if result == pgcr.Success {
					endTime := time.Now()
					log.Printf("[Malformed Worker] Added PGCR with instanceId %d (%d, %.0f, %.0f)", instanceId, i, endTime.Sub(startTime).Seconds(), lag.Seconds())
					break
				} else if result == pgcr.SystemDisabled {
					time.Sleep(30 * time.Second)
					i--
				}

				// Exponential Backoff
				time.Sleep(time.Duration(8 * i * i))
			}

		}(_instanceId)
	}
}
