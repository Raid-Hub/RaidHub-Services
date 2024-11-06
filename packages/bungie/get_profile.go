package bungie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type GetProfilesResponse struct {
	Response        DestinyProfileResponse `json:"Response"`
	ErrorCode       int                    `json:"ErrorCode"`
	ErrorStatus     string                 `json:"ErrorStatus"`
	ThrottleSeconds int                    `json:"ThrottleSeconds"`
}

type DestinyProfileResponse struct {
	Profile    SingleComponentResponseOfDestinyProfileComponent               `json:"profile"`
	Characters DictionaryComponentResponseOfint64AndDestinyCharacterComponent `json:"characters"`
}

type SingleComponentResponseOfDestinyProfileComponent struct {
	Data *DestinyProfileComponent `json:"data"`
}

type DestinyProfileComponent struct {
	UserInfo DestinyUserInfo `json:"userInfo"`
}

type DictionaryComponentResponseOfint64AndDestinyCharacterComponent struct {
	Data *map[int64]DestinyCharacterComponent `json:"data"`
}

func GetProfile(membershipType int, membershipId int64) (*DestinyProfileResponse, error) {
	url := fmt.Sprintf("%s/Platform/Destiny2/%d/Profile/%d/?components=100,200", getBungieURL(), membershipType, membershipId)
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

	var data GetProfilesResponse
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data.Response, nil
}
