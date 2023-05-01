package main

import (
	"log"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
)

func init() {
	config.Load("config.yaml")

	// Keep ws alive
	binance.WebsocketKeepalive = true
	binancesdk.WebsocketKeepalive = true
}

func main() {
	log.Println(config.Config)
	symbolPair := binancemexc.SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}
	binancemexc.ArbitrageManagerInstance = binancemexc.NewArbitrageManager(symbolPair)

	binancemexc.ArbitrageManagerInstance.Run()

	binancemexc.NewArbitrageTask(
		config.Config.BinanceApiKey,
		config.Config.BinanceSecretKey,
		config.Config.MexcApiKey,
		config.Config.MexcSecretKey,
		symbolPair,
		0.00005,
		0.00015,
	).Run()
}
