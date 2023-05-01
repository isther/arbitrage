package main

import (
	"log"

	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
)

func init() {
	config.Load("config.yaml")
}

func main() {
	log.Println(config.Config)
	symbolPair := binancemexc.SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}
	binancemexc.ArbitrageManagerInstance = binancemexc.NewArbitrageManager(symbolPair)

	go binancemexc.NewArbitrageTask(
		config.Config.BinanceApiKey,
		config.Config.BinanceSecretKey,
		config.Config.MexcApiKey,
		config.Config.MexcSecretKey,
		symbolPair,
		0.00005,
		0.00015,
	).Run()

	binancemexc.ArbitrageManagerInstance.StartWsBookTicker()
}
