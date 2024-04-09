package bungie

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	bungieURL = "/Platform/Destiny2/Stats/PostGameCarnageReport"
)

func GetPGCR(client *http.Client, baseURL string, instanceId int64, apiKey string) (*json.Decoder, int, func(), error) {
	instanceUrl := fmt.Sprintf("%s%s/%d/", baseURL, bungieURL, instanceId)
	req, _ := http.NewRequest("GET", instanceUrl, nil)
	req.Header.Set("X-API-KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, nil, err
	}

	decoder := json.NewDecoder(resp.Body)
	return decoder, resp.StatusCode, func() {
		resp.Body.Close()
	}, nil

}

// There are more fields here than recorded in this file, but these are the only ones we care about
type DestinyPostGameCarnageReportResponse struct {
	Response        DestinyPostGameCarnageReport `json:"Response"`
	ErrorCode       int                          `json:"ErrorCode"`
	ErrorStatus     string                       `json:"ErrorStatus"`
	ThrottleSeconds int                          `json:"ThrottleSeconds"`
}

type DestinyPostGameCarnageReport struct {
	ActivityDetails                 DestinyHistoricalStatsActivity      `json:"activityDetails"`
	Period                          string                              `json:"period"`
	StartingPhaseIndex              int                                 `json:"startingPhaseIndex"`
	ActivityWasStartedFromBeginning bool                                `json:"activityWasStartedFromBeginning"`
	Entries                         []DestinyPostGameCarnageReportEntry `json:"entries"`
}

type DestinyPostGameCarnageReportEntry struct {
	Player      DestinyPostGameCarnageReportPlayer       `json:"player"`
	CharacterId string                                   `json:"characterId"`
	Values      DestinyHistoricalStatsMap                `json:"values"`
	Extended    DestinyPostGameCarnageReportExtendedData `json:"extended"`
}

type DestinyPostGameCarnageReportPlayer struct {
	DestinyUserInfo DestinyUserInfo `json:"destinyUserInfo"`
	ClassHash       uint32          `json:"classHash"`
	CharacterClass  *string         `json:"characterClass"`
	RaceHash        uint32          `json:"raceHash"`
	GenderHash      uint32          `json:"genderHash"`
	CharacterLevel  int             `json:"characterLevel"`
	LightLevel      int             `json:"lightLevel"`
	EmblemHash      uint32          `json:"emblemHash"`
}

type DestinyPostGameCarnageReportExtendedData struct {
	Values  DestinyHistoricalStatsMap      `json:"values"`
	Weapons []DestinyHistoricalWeaponStats `json:"weapons"`
}

type DestinyHistoricalWeaponStats struct {
	ReferenceId uint32                    `json:"referenceId"`
	Values      DestinyHistoricalStatsMap `json:"values"`
}

type DestinyHistoricalStatsMap map[string]DestinyHistoricalStatsValue

type DestinyHistoricalStatsValue struct {
	Basic DestinyHistoricalStatsValuePair `json:"basic"`
}

type DestinyHistoricalStatsValuePair struct {
	Value        float32 `json:"value"`
	DisplayValue string  `json:"displayValue"`
}
