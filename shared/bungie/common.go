package bungie

import (
	"os"
	"sync"
)

var (
	bungieUrlBase string
	once          sync.Once
)

type BungieError struct {
	ErrorCode       int    `json:"ErrorCode"`
	Message         string `json:"Message"`
	ErrorStatus     string `json:"ErrorStatus"`
	ThrottleSeconds int    `json:"ThrottleSeconds"`
}

type DestinyUserInfo struct {
	IconPath                    *string `json:"iconPath"`
	MembershipType              int     `json:"membershipType"`
	MembershipId                string  `json:"membershipId"`
	DisplayName                 *string `json:"displayName"`
	BungieGlobalDisplayName     *string `json:"bungieGlobalDisplayName"`
	BungieGlobalDisplayNameCode *int    `json:"bungieGlobalDisplayNameCode"`
}

type DestinyHistoricalStatsActivity struct {
	InstanceId           string `json:"instanceId"`
	Mode                 int    `json:"mode"`
	Modes                []int  `json:"modes"`
	MembershipType       int    `json:"membershipType"`
	DirectorActivityHash uint32 `json:"directorActivityHash"`
}

type DestinyCharacterComponent struct {
	CharacterId    string `json:"characterId"`
	EmblemPath     string `json:"emblemPath"`
	EmblemHash     uint32 `json:"emblemHash"`
	ClassHash      uint32 `json:"classHash"`
	DateLastPlayed string `json:"dateLastPlayed"`
}

func getBungieURL() string {
	once.Do(func() {
		bungieUrlBase = os.Getenv("BUNGIE_URL_BASE")
		if bungieUrlBase == "" {
			bungieUrlBase = "https://www.bungie.net/"
		}
	})
	return bungieUrlBase
}
