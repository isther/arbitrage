package main

import (
	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
)

func init() {
	config.Load("config.yaml")
}

func main() {
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
		0.00007,
		0.00015,
	)

	binancemexc.ArbitrageManagerInstance.StartWsBookTicker()
}
