package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"raidhub/shared/monitoring"
	"raidhub/shared/pgcr"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Worker(wg *sync.WaitGroup, ch chan int64, offloadChannel chan int64, rabbitChannel *amqp.Channel, db *sql.DB) {
	defer wg.Done()

	securityKey := os.Getenv("BUNGIE_API_KEY")

	client := &http.Client{}

	randomVariation := retryDelayTime / 3

	for instanceID := range ch {
		startTime := time.Now()
		notFoundCount := 0
		errCount := 0
		i := 0

		for {
			reqStartTime := time.Now()
			result, activity, raw, err := pgcr.FetchAndProcessPGCR(client, instanceID, securityKey)
			if err != nil && result != pgcr.NotFound {
				log.Println(err)
			}

			statusStr := fmt.Sprintf("%d", result)
			attemptsStr := fmt.Sprintf("%d", i+1)

			monitoring.PGCRCrawlStatus.WithLabelValues(statusStr, attemptsStr).Inc()

			// Handle the result
			if result == pgcr.NonRaid {
				startDate, err := time.Parse(time.RFC3339, raw.Period)
				if err != nil {
					log.Fatal(err)
				}
				endDate := pgcr.CalculateDateCompleted(startDate, raw.Entries[0])

				lag := time.Since(endDate)
				monitoring.PGCRCrawlLag.WithLabelValues(statusStr, attemptsStr).Observe(float64(lag.Seconds()))
				break
			} else if result == pgcr.Success {
				lag, committed, err := pgcr.StorePGCR(activity, raw, db, rabbitChannel)
				if lag != nil {
					monitoring.PGCRCrawlLag.WithLabelValues(statusStr, attemptsStr).Observe(float64(lag.Seconds()))
				}
				if err != nil {
					errCount++
					log.Println(err)
					time.Sleep(5 * time.Second)
				} else {
					if committed {
						endTime := time.Now()
						workerTime := endTime.Sub(startTime).Milliseconds()
						reqTime := endTime.Sub(reqStartTime)
						log.Printf("Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime.Milliseconds(), lag.Seconds())
					}
					break
				}
			} else if result == pgcr.InternalError {
				errCount++
				time.Sleep(10 * time.Second)
			} else if result == pgcr.NotFound {
				notFoundCount++
			} else if result == pgcr.SystemDisabled {
				monitoring.PGCRCrawlLag.WithLabelValues(statusStr, attemptsStr).Observe(0)
				time.Sleep(45 * time.Second)
				continue
			} else if result == pgcr.InsufficientPrivileges {
				go logMissedInstance(instanceID, startTime)
				logInsufficentPrivileges(instanceID)
				pgcr.WriteMissedLog(instanceID)
				break
			} else if result == pgcr.BadFormat {
				pgcr.WriteMissedLog(instanceID)
				offloadChannel <- instanceID
				break
			}

			// If we have not found the instance id after some time
			if notFoundCount > 4 || errCount > 3 {
				pgcr.WriteMissedLog(instanceID)
				offloadChannel <- instanceID
				break
			}

			timeout := time.Duration((retryDelayTime - randomVariation + rand.Intn(retryDelayTime*(i+1)))) * time.Millisecond
			time.Sleep(timeout)
			i++
		}
	}
}
