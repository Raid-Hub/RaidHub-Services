package bungie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"raidhub/shared/monitoring"
	"time"
)

type LinkedProfilesBungieResponse struct {
	Response        LinkedProfilesResponse `json:"Response"`
	ErrorCode       int                    `json:"ErrorCode"`
	ErrorStatus     string                 `json:"ErrorStatus"`
	ThrottleSeconds int                    `json:"ThrottleSeconds"`
}

type LinkedProfilesResponse struct {
	Profiles []DestinyProfileUserInfoCard `json:"profiles"`
}

type DestinyProfileUserInfoCard struct {
	IconPath                    *string `json:"iconPath"`
	MembershipType              int     `json:"membershipType"`
	MembershipId                string  `json:"membershipId"`
	DisplayName                 *string `json:"displayName"`
	BungieGlobalDisplayName     *string `json:"bungieGlobalDisplayName"`
	BungieGlobalDisplayNameCode *int    `json:"bungieGlobalDisplayNameCode"`
	DateLastPlayed              string  `json:"dateLastPlayed"`
}

func GetLinkedProfiles(membershipType int, membershipId string) ([]DestinyProfileUserInfoCard, error) {
	url := fmt.Sprintf("https://www.bungie.net/Platform/Destiny2/%d/Profile/%s/LinkedProfiles/?getAllMemberships=true", membershipType, membershipId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []DestinyProfileUserInfoCard{}, err
	}

	apiKey := os.Getenv("BUNGIE_API_KEY") // Read the API key from the BUNGIE_API_KEY environment variable
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []DestinyProfileUserInfoCard{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var data BungieError
		if err := decoder.Decode(&data); err != nil {
			return []DestinyProfileUserInfoCard{}, err
		}
		monitoring.BungieErrorCode.WithLabelValues(data.ErrorStatus).Inc()

		defer func() {
			if data.ThrottleSeconds > 0 {
				time.Sleep(time.Duration(data.ThrottleSeconds) * time.Second)
			}
		}()

		return []DestinyProfileUserInfoCard{}, fmt.Errorf("error response: %s (%d)", data.Message, data.ErrorCode)
	}

	var data LinkedProfilesBungieResponse
	if err := decoder.Decode(&data); err != nil {
		return []DestinyProfileUserInfoCard{}, err
	}

	return data.Response.Profiles, nil
}