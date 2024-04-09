package bungie

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

const concurrentPages = 5

func GetActivityHistory(membershipType int, membershipId string, characterId string, out chan int64) {
	ch := make(chan int)
	open := true
	go func() {
		i := 0
		for open {
			ch <- i
			i++
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < concurrentPages; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for page := range ch {
				results, err := getActivityHistoryPage(membershipType, membershipId, characterId, page)
				if err != nil {
					log.Printf("Error fetching activity history page: %s", err)
					break
				}

				if len(results) == 0 {
					break
				}

				for _, activity := range results {
					instanceId, err := strconv.ParseInt(activity.ActivityDetails.InstanceId, 10, 64)
					if err != nil {
						log.Printf("Error parsing instance id: %s", err)
						continue
					}
					out <- instanceId
				}
			}
		}()
	}

	wg.Wait()
	open = false
}

func getActivityHistoryPage(membershipType int, membershipId string, characterId string, page int) ([]DestinyHistoricalStatsPeriodGroup, error) {
	log.Printf("Getting /Destiny2/%d/Account/%s/Character/%s/ page=%d", membershipType, membershipId, characterId, page)
	url := fmt.Sprintf("https://www.bungie.net/Platform/Destiny2/%d/Account/%s/Character/%s/Stats/Activities/?mode=4&count=250&page=%d", membershipType, membershipId, characterId, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, err
	}

	apiKey := os.Getenv("BUNGIE_API_KEY")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var data BungieError
		if err := decoder.Decode(&data); err != nil {
			return []DestinyHistoricalStatsPeriodGroup{}, err
		}

		defer func() {
			if data.ThrottleSeconds > 0 {
				time.Sleep(time.Duration(data.ThrottleSeconds) * time.Second)
			}
		}()

		return []DestinyHistoricalStatsPeriodGroup{}, fmt.Errorf("error response: %s (%d)", data.Message, data.ErrorCode)
	}

	var data ActivityHistoryResponse
	if err := decoder.Decode(&data); err != nil {
		return []DestinyHistoricalStatsPeriodGroup{}, err
	}

	return data.Response.Activities, nil
}
