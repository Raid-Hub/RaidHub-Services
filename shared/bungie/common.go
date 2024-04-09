package bungie

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
