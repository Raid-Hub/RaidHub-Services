package pgcr

// There are more fields here than recorded in this file, but these are the only ones we care about

type DestinyPostGameCarnageReportErrorCode struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

type DestinyPostGameCarnageReportResponse struct {
	Response DestinyPostGameCarnageReport `json:"Response"`
}

type DestinyPostGameCarnageReport struct {
	ActivityDetails                 DestinyHistoricalStatsActivity      `json:"activityDetails"`
	Period                          string                              `json:"period"`
	StartingPhaseIndex              int                                 `json:"startingPhaseIndex"`
	ActivityWasStartedFromBeginning bool                                `json:"activityWasStartedFromBeginning"`
	Entries                         []DestinyPostGameCarnageReportEntry `json:"entries"`
}
type DestinyHistoricalStatsActivity struct {
	InstanceId           string `json:"instanceId"`
	Mode                 int    `json:"mode"`
	Modes                []int  `json:"modes"`
	MembershipType       int    `json:"membershipType"`
	DirectorActivityHash uint32 `json:"directorActivityHash"`
}

type DestinyPostGameCarnageReportEntry struct {
	Player      Player                                   `json:"player"`
	CharacterId string                                   `json:"characterId"`
	Values      DestinyHistoricalStatsMap                `json:"values"`
	Extended    DestinyPostGameCarnageReportExtendedData `json:"extended"`
}

type Player struct {
	DestinyUserInfo DestinyUserInfo `json:"destinyUserInfo"`
	ClassHash       uint32          `json:"classHash"`
	CharacterClass  *string         `json:"characterClass"`
	RaceHash        uint32          `json:"raceHash"`
	GenderHash      uint32          `json:"genderHash"`
	CharacterLevel  int             `json:"characterLevel"`
	LightLevel      int             `json:"lightLevel"`
	EmblemHash      uint32          `json:"emblemHash"`
}

type DestinyUserInfo struct {
	IconPath                    *string `json:"iconPath"`
	CrossSaveOverride           int     `json:"crossSaveOverride"`
	ApplicableMembershipTypes   []int   `json:"applicableMembershipTypes"`
	MembershipType              int     `json:"membershipType"`
	MembershipId                string  `json:"membershipId"`
	DisplayName                 *string `json:"displayName"`
	BungieGlobalDisplayName     *string `json:"bungieGlobalDisplayName"`
	BungieGlobalDisplayNameCode *int    `json:"bungieGlobalDisplayNameCode"`
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
