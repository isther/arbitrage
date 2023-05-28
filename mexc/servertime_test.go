package mexc

import (
	"context"
	"testing"

	"github.com/isther/arbitrage/config"
)

func TestServerTime(t *testing.T) {
	config.Load("../config.yaml")

	servertime, err := NewClient(config.Config.MexcApiKey, config.Config.MexcSecretKey).NewServerTimeService().Do(context.Background())
	if err != nil {
		panic(err)
	}

	t.Log(servertime)
}
