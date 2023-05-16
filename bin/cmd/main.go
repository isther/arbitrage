package main

import (
	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
	"github.com/sirupsen/logrus"
)

var (
	symbolPair = binancemexc.SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}

	arbitrageManager = binancemexc.NewArbitrageManager(symbolPair)
	orderManager     = binancemexc.NewOrderManager()
)

func init() {
	config.Load("config.yaml")

	// Keep ws alive
	binance.WebsocketKeepalive = true
	binancesdk.WebsocketKeepalive = true
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	arbitrageManager.Run(orderManager.OpenBinanceOrderIdCh, orderManager.CloseBinanceOrderIdCh)
	orderManager.Run()
}

func main() {
	arbitrageManager.StartTask(
		binancemexc.NewArbitrageTask(
			config.Config.BinanceApiKey,
			config.Config.BinanceSecretKey,
			config.Config.MexcApiKey,
			config.Config.MexcSecretKey,
			symbolPair,
			config.Config.ProfitRatio,
			config.Config.MinRatio,
			config.Config.MaxRatio,
		),
		orderManager.OpenMexcOrderIdCh,
		orderManager.CloseMexcOrderIdCh,
	)
}
