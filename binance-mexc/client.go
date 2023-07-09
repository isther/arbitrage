package binancemexc

import (
	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
)

func newMexcClient() *mexc.Client {
	return mexc.NewClient(config.Config.MexcApiKey, config.Config.MexcSecretKey)
}

func newBinanceClient() *binancesdk.Client {
	return binancesdk.NewClient(config.Config.BinanceApiKey, config.Config.BinanceSecretKey)
}

func newBinanceFuturesClient() *futures.Client {
	return futures.NewClient(config.Config.BinanceApiKey, config.Config.BinanceSecretKey)
}
