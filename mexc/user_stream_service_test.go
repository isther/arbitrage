package mexc

import (
	"context"
	"fmt"
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

	fmt.Println(listenKey)

	err = client.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(context.Background())
	assert.Nil(err)

	err = client.NewCloseUserStreamService().ListenKey(listenKey).Do(context.Background())
	assert.Nil(err)
}
