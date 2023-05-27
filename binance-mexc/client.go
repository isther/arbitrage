package binancemexc

import (
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
)

func newMexcClient() *mexc.Client {
	return mexc.NewClient(config.Config.MexcApiKey, config.Config.MexcSecretKey)
}
