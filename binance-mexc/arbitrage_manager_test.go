package binancemexc

import (
	"testing"
	"time"

	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/config"
)

func TestArbitrageManager(t *testing.T) {
	config.Load("../config.yaml")
	symbolPair := SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}
	arbitrageManager := NewArbitrageManager(symbolPair)
	arbitrageManager.Run()

	time.Sleep(2 * time.Second)
	params := binance.NewOrderTradeParmes(config.Config.BinanceApiKey).
		Symbol("BTCUSDT").Side(binance.SideTypeBuy).OrderType(binance.OrderTypeLimit).
		NewOrderRespType(binance.NewOrderRespTypeACK).TimeInForce(binance.TimeInForceTypeGTC).
		Price("30000.000000").Quantity("0.0005101").
		Signature(config.Config.BinanceSecretKey)

	arbitrageManager.websocketApiServiceManager.Send(binance.NewOrderTrade(params))
	time.Sleep(5 * time.Second)
}
