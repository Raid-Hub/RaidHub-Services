package pgcr

import (
	"database/sql"
	"log"
	"net/http"
	"raidhub/shared/bungie"
	"raidhub/shared/monitoring"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PGCRResult int

const (
	Success                PGCRResult = 0
	AlreadyExists          PGCRResult = 1
	NonRaid                PGCRResult = 2
	NotFound               PGCRResult = 3
	SystemDisabled         PGCRResult = 4
	InsufficientPrivileges PGCRResult = 5
	BadFormat              PGCRResult = 6
	InternalError          PGCRResult = 7
)

func FetchAndStorePGCR(client *http.Client, instanceID int64, db *sql.DB, channel *amqp.Channel, baseURL string, apiKey string) (PGCRResult, *time.Duration) {
	decoder, statusCode, cleanup, err := bungie.GetPGCR(client, baseURL, instanceID, apiKey)
	if err != nil {
		log.Printf("Error fetching instanceId %d: %s", instanceID, err)
		return InternalError, nil
	}
	defer cleanup()

	if statusCode != http.StatusOK {
		var data bungie.BungieError
		if err := decoder.Decode(&data); err != nil {
			log.Printf("Error decoding response for instanceId %d: %s", instanceID, err)
			return BadFormat, nil
		}
		monitoring.BungieErrorCode.WithLabelValues(data.ErrorStatus).Inc()

		defer func() {
			if data.ThrottleSeconds > 0 {
				log.Printf("Throttled: %d seconds", data.ThrottleSeconds)
				time.Sleep(time.Duration(data.ThrottleSeconds) * time.Second)
			}
		}()

		if data.ErrorCode == 1653 {
			// PGCRNotFound
			return NotFound, nil
		}

		log.Printf("Error response for instanceId %d: %s (%d)", instanceID, data.Message, data.ErrorCode)
		if data.ErrorCode == 5 {
			// SystemDisabled
			time.Sleep(30 * time.Second)
			return SystemDisabled, nil
		} else if data.ErrorCode == 1672 {
			// BabelTimeout
			time.Sleep(5 * time.Second)
			return NotFound, nil
		} else if data.ErrorCode == 12 {
			// InsufficientPrivileges, redacted
			return InsufficientPrivileges, nil
		}

		return BadFormat, nil
	}

	var data bungie.DestinyPostGameCarnageReportResponse
	if err := decoder.Decode(&data); err != nil {
		log.Printf("Error decoding response for instanceId %d: %s", instanceID, err)
		return BadFormat, nil
	}
	monitoring.BungieErrorCode.WithLabelValues(data.ErrorStatus).Inc()

	if data.Response.ActivityDetails.Mode != 4 {
		// Skip non raid
		startDate, err := time.Parse(time.RFC3339, data.Response.Period)
		if err != nil {
			log.Println(err)
			return InternalError, nil
		}
		endDate := CalculateDateCompleted(startDate, data.Response.Entries[0])

		lag := time.Since(endDate)

		return NonRaid, &lag
	}

	pgcr, err := ProcessDestinyReport(&data.Response)
	if err != nil {
		return BadFormat, nil
	}

	lag, committed, err := StorePGCR(pgcr, &data.Response, db, channel)
	if err != nil {
		log.Println(err)
		return InternalError, nil
	} else if committed {
		return Success, lag
	} else {
		return AlreadyExists, lag
	}
}
