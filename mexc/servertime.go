package mexc

import (
	"encoding/json"

	"github.com/go-resty/resty/v2"
)

func ServerTime() int64 {
	var m = make(map[string]int64)

	resp := PublicGet(getHttpEndpoint()+getServerTimeApi(), "")

	servertime, ok := resp.(*resty.Response)
	if !ok {
		return 0
	}
	if err := json.Unmarshal(servertime.Body(), &m); err != nil {
		return 0
	}

	return m["serverTime"]
}
