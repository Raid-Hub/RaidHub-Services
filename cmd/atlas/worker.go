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

func Worker(wg *sync.WaitGroup, ch chan int64, failuresChannel chan int64, offloadChannel chan int64, rabbitChannel *amqp.Channel, db *sql.DB) {
	defer wg.Done()

	securityKey := os.Getenv("BUNGIE_API_KEY")
	proxy := os.Getenv("PGCR_URL_BASE")

	client := &http.Client{}

	randomVariation := retryDelayTime / 3

	for instanceID := range ch {
		startTime := time.Now()
		notFoundCount := 0
		errCount := 0
		i := 0
		var result pgcr.PGCRResult
		var lag *time.Duration
		var reqTime time.Duration

		var statusStr string
		var attemptsStr string
		for {
			reqStartTime := time.Now()
			result, lag = pgcr.FetchAndStorePGCR(client, instanceID, db, rabbitChannel, proxy, securityKey)

			statusStr = fmt.Sprintf("%d", result)
			attemptsStr = fmt.Sprintf("%d", i+1)

			monitoring.PGCRCrawlStatus.WithLabelValues(statusStr, attemptsStr).Inc()
			if lag != nil {
				monitoring.PGCRCrawlLag.WithLabelValues(statusStr, attemptsStr).Observe(float64(lag.Seconds()))
			}

			// Handle the result
			if result == pgcr.AlreadyExists || result == pgcr.NonRaid {
				break
			}
			if result == pgcr.Success {
				endTime := time.Now()
				workerTime := endTime.Sub(startTime).Milliseconds()
				reqTime = endTime.Sub(reqStartTime)
				log.Printf("Added PGCR with instanceId %d (%d / %d / %d / %.0f)", instanceID, i, workerTime, reqTime.Milliseconds(), lag.Seconds())
				break
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
				failuresChannel <- instanceID
				logMissedInstance(instanceID, startTime, false)
				logInsufficentPrivileges(instanceID)
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
