package bungie

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type ActivityHistoryResponse struct {
	Response        DestinyActivityHistoryResults `json:"Response"`
	ErrorCode       int                           `json:"ErrorCode"`
	ErrorStatus     string                        `json:"ErrorStatus"`
	ThrottleSeconds int                           `json:"ThrottleSeconds"`
}

type DestinyActivityHistoryResults struct {
	Activities []DestinyHistoricalStatsPeriodGroup `json:"activities"`
}

type DestinyHistoricalStatsPeriodGroup struct {
	ActivityDetails DestinyHistoricalStatsActivity `json:"activityDetails"`
}

func GetActivityHistory(membershipType int, membershipId int64, characterId int64, concurrentPages int, out chan int64) error {
	ch := make(chan int)

	results, _, err := GetActivityHistoryPage(membershipType, membershipId, characterId, 0)
	if err != nil {
		return err
	}

	for _, activity := range results {
		out <- activity.ActivityDetails.InstanceId
	}

	open := true
	go func() {
		i := 1
		for open {
			ch <- i
			i++
		}
	}()

	var wg sync.WaitGroup
	for j := 0; j < concurrentPages; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for page := range ch {
				results, _, err := GetActivityHistoryPage(membershipType, membershipId, characterId, page)
				if err != nil {
					log.Printf("Error fetching activity history page: %s", err)
				}

				if len(results) == 0 {
					break
				}

				for _, activity := range results {
					out <- activity.ActivityDetails.InstanceId
				}
			}
		}()
	}

	wg.Wait()
	open = false
	return nil
}

func GetActivityHistoryPage(membershipType int, membershipId int64, characterId int64, page int) ([]DestinyHistoricalStatsPeriodGroup, int, error) {
	log.Printf("Getting /Destiny2/%d/Account/%d/Character/%d/ page=%d", membershipType, membershipId, characterId, page)
	url := fmt.Sprintf("%s/Platform/Destiny2/%d/Account/%d/Character/%d/Stats/Activities/?mode=4&count=250&page=%d", getBungieURL(), membershipType, membershipId, characterId, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, 0, err
	}

	apiKey := os.Getenv("BUNGIE_API_KEY")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, 0, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var data BungieError
		if err := decoder.Decode(&data); err != nil {
			return []DestinyHistoricalStatsPeriodGroup{}, 0, err
		}

		defer func() {
			if data.ThrottleSeconds > 0 {
				time.Sleep(time.Duration(data.ThrottleSeconds) * time.Second)
			}
		}()

		return []DestinyHistoricalStatsPeriodGroup{}, data.ErrorCode, fmt.Errorf("error response: %s (%d)", data.Message, data.ErrorCode)
	}

	var data ActivityHistoryResponse
	if err := decoder.Decode(&data); err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, 0, err
	}

	return data.Response.Activities, data.ErrorCode, nil
}
