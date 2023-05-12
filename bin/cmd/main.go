package main

import (
	"log"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
)

var (
	symbolPair = binancemexc.SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}

	arbitrageManager = binancemexc.NewArbitrageManager(symbolPair)
)

func init() {
	config.Load("config.yaml")

	// Keep ws alive
	binance.WebsocketKeepalive = true
	binancesdk.WebsocketKeepalive = true
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	arbitrageManager.Run()
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
	)
}
