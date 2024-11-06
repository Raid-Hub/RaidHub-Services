package bungie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type GetGroupResponse struct {
	Response        GroupResponse `json:"Response"`
	ErrorCode       int           `json:"ErrorCode"`
	ErrorStatus     string        `json:"ErrorStatus"`
	ThrottleSeconds int           `json:"ThrottleSeconds"`
}

type GroupResponse struct {
	Detail GroupV2 `json:"detail"`
}

func GetGroup(groupId int64) (*GroupResponse, error) {
	url := fmt.Sprintf("%s/Platform/GroupV2/%d", getBungieURL(), groupId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	apiKey := os.Getenv("BUNGIE_API_KEY") // Read the API key from the BUNGIE_API_KEY environment variable
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var data BungieError
		if err := decoder.Decode(&data); err != nil {
			return nil, err
		}

		defer func() {
			if data.ThrottleSeconds > 0 {
				time.Sleep(time.Duration(data.ThrottleSeconds) * time.Second)
			}
		}()

		return nil, fmt.Errorf("error response: %s (%d)", data.Message, data.ErrorCode)
	}

	var data GetGroupResponse
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data.Response, nil
}
