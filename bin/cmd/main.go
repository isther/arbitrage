package main

import (
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	binancemexc "github.com/isther/arbitrage/binance-mexc"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/dingding"
	"github.com/sirupsen/logrus"
)

var (
	symbolPair = binancemexc.SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}

	arbitrageManager = binancemexc.NewArbitrageManager(symbolPair)
	account          = binancemexc.NewAccount(symbolPair)
)

func init() {
	config.Load("config.yaml")

	// Config
	// Keep ws alive
	binance.WebsocketKeepalive = true
	binancesdk.WebsocketKeepalive = true

	// Add dingding bot hook
	logrus.AddHook(dingding.NewDingDingBotHook(
		config.Config.LogDingDingConfig.AccessToken, config.Config.LogDingDingConfig.Secrect,
		config.Config.ErrorDingDingConfig.AccessToken, config.Config.ErrorDingDingConfig.Secrect,
		10000,
	))

	// Set log format
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	// Start
	go binancemexc.StartCalculateKline()
	arbitrageManager.Start()
	account.Start()

	logrus.Info("启动中...")
	time.Sleep(1 * time.Second)
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
		account.OrderIDsCh,
	)
}
