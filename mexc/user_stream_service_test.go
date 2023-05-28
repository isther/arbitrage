package mexc

import (
	"context"
	"testing"

	"github.com/isther/arbitrage/config"
	"github.com/stretchr/testify/assert"
)

func TestListenKey(t *testing.T) {
	assert := assert.New(t)
	config.Load("../config.yaml")
	client := NewClient(config.Config.MexcApiKey, config.Config.MexcSecretKey)

	listenKey, err := client.NewStartUserStreamService().Do(context.Background())
	assert.Nil(err)

	t.Log(listenKey)

	err = client.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(context.Background())
	assert.Nil(err)

	err = client.NewCloseUserStreamService().ListenKey(listenKey).Do(context.Background())
	assert.Nil(err)
}
