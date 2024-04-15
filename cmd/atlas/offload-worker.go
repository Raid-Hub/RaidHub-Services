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

func offloadWorker(ch chan int64, failuresChannel chan int64, rabbitChannel *amqp.Channel, db *sql.DB) {
	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	for id := range ch {
		// Spawn a worker for each instanceId
		go func(instanceId int64) {
			log.Printf("Offloading instanceId %d", instanceId)
			startTime := time.Now()
			for i := 1; i <= 5; i++ {
				result, lag := pgcr.FetchAndStorePGCR(client, instanceId, db, rabbitChannel, proxy, securityKey)

				if result == pgcr.AlreadyExists || result == pgcr.NonRaid {
					return
				} else if result == pgcr.Success {
					endTime := time.Now()
					log.Printf("[Offload Worker] Added PGCR with instanceId %d (%d, %.0f, %.0f)", instanceId, i, endTime.Sub(startTime).Seconds(), lag.Seconds())
					return
				} else if result == pgcr.SystemDisabled {
					i--
					time.Sleep(60 * time.Second)
					continue
				}

				if i == 3 {
					logMissedInstanceWarning(instanceId, startTime)
				}

				// Exponential Backoff
				time.Sleep(time.Duration(8*i*i) * time.Second)
			}
			logMissedInstance(instanceId, startTime, false)
			failuresChannel <- instanceId

		}(id)
	}
}
