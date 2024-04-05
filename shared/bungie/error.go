package bungie

type BungieError struct {
	ErrorCode       int    `json:"ErrorCode"`
	Message         string `json:"Message"`
	ErrorStatus     string `json:"ErrorStatus"`
	ThrottleSeconds int    `json:"ThrottleSeconds"`
}
