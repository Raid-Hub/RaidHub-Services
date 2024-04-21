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

	client := &http.Client{}

	for id := range ch {
		// Spawn a worker for each instanceId
		go func(instanceId int64) {
			log.Printf("Offloading instanceId %d", instanceId)
			startTime := time.Now()
			for i := 1; i <= 5; i++ {
				result, activity, raw, err := pgcr.FetchAndProcessPGCR(client, instanceId, securityKey)

				if err != nil {
					log.Println(err)
				}

				if result == pgcr.NonRaid {
					log.Printf("[Offload Worker] Found non-raid raid with instanceId %d", instanceId)
					return
				} else if result == pgcr.Success {
					lag, committed, err := pgcr.StorePGCR(activity, raw, db, rabbitChannel)
					endTime := time.Now()
					if err != nil {
						log.Println(err)
					} else if committed {
						log.Printf("[Offload Worker] Added PGCR with instanceId %d (%d, %.0f, %.0f)", instanceId, i, endTime.Sub(startTime).Seconds(), lag.Seconds())
						return
					} else {
						log.Printf("[Offload Worker] Found duplicate raid with instanceId %d (%d, %.0f, %.0f)", instanceId, i, endTime.Sub(startTime).Seconds(), lag.Seconds())
					}
				} else if result == pgcr.SystemDisabled {
					i--
					time.Sleep(60 * time.Second)
					continue
				}

				if i == 3 {
					logMissedInstanceWarning(instanceId, startTime)
				}

				// Exponential Backoff
				time.Sleep(time.Duration(10*i*i) * time.Second)
			}
			logMissedInstance(instanceId, startTime, false)
			failuresChannel <- instanceId

		}(id)
	}
}
