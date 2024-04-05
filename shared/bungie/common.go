package bungie

type BungieError struct {
	ErrorCode       int    `json:"ErrorCode"`
	Message         string `json:"Message"`
	ErrorStatus     string `json:"ErrorStatus"`
	ThrottleSeconds int    `json:"ThrottleSeconds"`
}

type UserInfoCard struct {
	IconPath                    *string `json:"iconPath"`
	MembershipType              int     `json:"membershipType"`
	MembershipId                string  `json:"membershipId"`
	DisplayName                 *string `json:"displayName"`
	BungieGlobalDisplayName     *string `json:"bungieGlobalDisplayName"`
	BungieGlobalDisplayNameCode *int    `json:"bungieGlobalDisplayNameCode"`
}
