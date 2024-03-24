package pgcr

import (
	"fmt"
	"net/http"
)

const (
	bungieURL = "/Platform/Destiny2/Stats/PostGameCarnageReport"
)

func getPGCR(client *http.Client, baseURL string, instanceId int64, apiKey string) (*http.Response, error) {
	instanceUrl := fmt.Sprintf("%s%s/%d/", baseURL, bungieURL, instanceId)
	req, _ := http.NewRequest("GET", instanceUrl, nil)
	req.Header.Set("X-API-KEY", apiKey)

	return client.Do(req)
}
