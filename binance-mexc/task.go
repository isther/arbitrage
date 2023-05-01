package binancemexc

import (
	"log"
	"os"
	"sync"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
	"github.com/shopspring/decimal"
)

type Task struct {
	binanceApiKey    string
	binanceSecretKey string

	mexcApiKey    string
	mexcSecretKey string

	symbolPairs SymbolPair

	binanceSymbolEvent    *binancesdk.WsBookTickerEvent
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent
	mexcSymbolEvent       *mexc.WsBookTickerEvent

	MinMinRatio decimal.Decimal
	MinRatio    decimal.Decimal
	MaxRatio    decimal.Decimal

	stopCh chan struct{}
	L      sync.RWMutex
}

func NewArbitrageTask(
	binanceApiKey,
	binanceSecretKey,
	mexcApiKey,
	mexcSecretKey string,
	symbolPairs SymbolPair,
	minRatio, maxRatio float64) *Task {
	return &Task{
		binanceApiKey:    binanceApiKey,
		binanceSecretKey: binanceSecretKey,
		mexcApiKey:       mexcApiKey,
		mexcSecretKey:    mexcSecretKey,
		symbolPairs:      symbolPairs,
		stopCh:           make(chan struct{}),
		MinRatio:         decimal.NewFromFloat(minRatio),
		MaxRatio:         decimal.NewFromFloat(maxRatio),
	}
}

func (t *Task) Run() {
	for {
		select {
		case binanceSymbolEvent := <-ArbitrageManagerInstance.binanceSymbolEventCh:
			go func() {
				if t.L.TryLock() {
					defer t.L.Unlock()
					t.binanceSymbolEvent = binanceSymbolEvent
					t.Trade()
				} else {
					log.Println("Wait")
				}
			}()
		case stableCoinSymbolEvent := <-ArbitrageManagerInstance.stableCoinSymbolEventCh:
			go func() {
				if t.L.TryLock() {
					defer t.L.Unlock()
					t.stableCoinSymbolEvent = stableCoinSymbolEvent
					t.Trade()
				} else {
					log.Println("Wait")
				}
			}()
		case mexcSymbolEvent := <-ArbitrageManagerInstance.mexcSymbolEventCh:
			go func() {
				if t.L.TryLock() {
					defer t.L.Unlock()
					t.mexcSymbolEvent = mexcSymbolEvent
					t.Trade()
				} else {
					log.Println("Wait")
				}
			}()
		case <-t.stopCh:
			log.Println("Stop")
			return
		}
	}
}

func (t *Task) Trade() {
	stableEvent, binanceEvent := t.stableCoinSymbolEvent, t.binanceSymbolEvent
	mexcEvent := t.mexcSymbolEvent
	if stableEvent == nil || binanceEvent == nil || mexcEvent == nil {
		log.Println("Get nil event")
		return
	}
	if mexcEvent.Data.AskPrice == "" {
		log.Println("Get null mexc event")
		return
	}

	// Price
	// AskPrice: Minimum sell pice | 卖1
	// BidPrice: Maximum buy price | 买1

	// （1除以tusd/usdt区的卖1价）减去（btc/tusd区卖1价除以btc/usdt区买1价）大于万0.7 小于万1.5
	stableSymbolAskPrice, _ := decimal.NewFromString(stableEvent.BestAskPrice)
	binanceSymbolAskPrice, _ := decimal.NewFromString(binanceEvent.BestAskPrice)
	mexcSymbolBidPrice, _ := decimal.NewFromString(mexcEvent.Data.BidPrice)
	if t.judgeRatio(false, binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolAskPrice) {
		log.Println("Do mode 2")
		var (
			doneBinanceCh = make(chan struct{})
			doneMexcCh    = make(chan struct{})
		)
		// 币安把TUSD买入为BTC、抹茶把BTC卖出为USDT；
		stableSymbolAskQty, _ := decimal.NewFromString(stableEvent.BestAskQty)   // TUSDUSDT
		binanceSymbolAskQty, _ := decimal.NewFromString(binanceEvent.BestAskQty) // BTCTUSD
		mexcSymbolBidQty, _ := decimal.NewFromString(mexcEvent.Data.BidQty)      // BTCUSDT

		stabldToAQty := stableSymbolAskQty
		binanceToAQty := binanceSymbolAskQty.Mul(binanceSymbolAskPrice)
		mexcToAQty := mexcSymbolBidQty.Mul(mexcSymbolBidPrice).Div(stableSymbolAskPrice)

		// Quantity
		aQty := decimal.Min(stabldToAQty, binanceToAQty, mexcToAQty, decimal.NewFromFloat(11.0))
		aQty = decimal.Max(aQty, decimal.NewFromFloat(11.0))

		// Trade binance
		go func() {
			requestCh <- t.getOrderBinanceTrade(
				t.symbolPairs.BinanceSymbol,
				binance.SideTypeBuy,
				aQty.Div(binanceSymbolAskPrice).String(),
			)

			doneBinanceCh <- struct{}{}
		}()

		// Trade mexc
		go func() {
			res, err := mexc.MexcBTCSell(
				config.Config.MexcCookie,
				mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(),
				aQty.Mul(stableSymbolAskPrice).Div(mexcSymbolBidPrice).String(),
			)
			if err != nil {
				log.Println("mexc trade error", err)
			}

			log.Println("mexc trade", res)
			doneMexcCh <- struct{}{}
		}()

		<-doneBinanceCh
		<-doneMexcCh
		os.Exit(0)
	}
	// （tusd/usdt区的买1价）减去（btc/usdt区卖1价除以btc/tusd区买1价）大于万0.7 小于万1.5
	stableSymbolBidPrice, _ := decimal.NewFromString(stableEvent.BestBidPrice)
	mexcSymbolAskPrice, _ := decimal.NewFromString(mexcEvent.Data.AskPrice)
	binanceSymbolBidPrice, _ := decimal.NewFromString(binanceEvent.BestBidPrice)
	if t.judgeRatio(true, binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolBidPrice) {
		log.Println("Do mode 1")
		var (
			doneBinanceCh = make(chan struct{})
			doneMexcCh    = make(chan struct{})
		)
		// 币安把BTC卖出为TUSD、抹茶把USDT买入为BTC；
		stableSymbolBidQty, _ := decimal.NewFromString(stableEvent.BestBidQty)   // TUSDUSDT
		mexcSymbolAskQty, _ := decimal.NewFromString(mexcEvent.Data.AskQty)      // BTCUSDT
		binanceSymbolBidQty, _ := decimal.NewFromString(binanceEvent.BestBidQty) //BTCTUSD

		stabldToAQty := stableSymbolBidQty
		binanceToAQty := binanceSymbolBidQty.Mul(binanceSymbolBidPrice)
		mexcToAQty := mexcSymbolAskQty.Mul(mexcSymbolAskPrice).Div(stableSymbolBidPrice)

		// Quantity
		aQty := decimal.Min(stabldToAQty, binanceToAQty, mexcToAQty, decimal.NewFromFloat(11.0))
		aQty = decimal.Max(aQty, decimal.NewFromFloat(11.0))

		// Trade binance
		go func() {
			requestCh <- t.getOrderBinanceTrade(
				t.symbolPairs.BinanceSymbol,
				binance.SideTypeSell,
				aQty.Div(binanceSymbolBidPrice).String(),
			)

			doneBinanceCh <- struct{}{}
		}()

		// Trade mexc
		go func() {
			res, err := mexc.MexcBTCBuy(
				config.Config.MexcCookie,
				mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(),
				aQty.Mul(stableSymbolBidPrice).Div(mexcSymbolAskPrice).String(),
			)
			if err != nil {
				log.Println("mexc trade error", err)
			}

			log.Println("mexc trade", res)
			doneMexcCh <- struct{}{}

		}()

		<-doneBinanceCh
		<-doneMexcCh
		os.Exit(0)
	}
}

// forward and reverse
func (t *Task) judgeRatio(reverse bool, taPrice, tbPrice, stableSymbolPrice decimal.Decimal) bool {
	var (
		ratio decimal.Decimal
	)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	if !reverse {
		ratio = decimal.NewFromFloat32(1).Div(stableSymbolPrice).
			Sub(
				taPrice.Div(
					tbPrice,
				),
			)
			// if 0 <= ratio.Cmp(t.MinRatio) && ratio.Cmp(t.MaxRatio) <= 0 {
		log.Println(
			"[Mode2]",
			"TUSD/USDT: ", stableSymbolPrice,
			"BTC/TUSD: ", taPrice,
			"BTC/USDT: ", tbPrice,
			"Ratio:", ratio,
		)
		// }
	} else {
		ratio = stableSymbolPrice.
			Sub(
				tbPrice.Div(
					taPrice,
				),
			)

			// if 0 <= ratio.Cmp(t.MinRatio) && ratio.Cmp(t.MaxRatio) <= 0 {
		log.Println(
			"[Mode1]",
			"TUSD/USDT: ", stableSymbolPrice,
			"BTC/TUSD: ", taPrice,
			"BTC/USDT: ", tbPrice,
			"Ratio:", ratio,
		)
		// }
	}

	return 0 <= ratio.Cmp(t.MinRatio) && ratio.Cmp(t.MaxRatio) <= 0
}

func (t *Task) getOrderBinanceTrade(symbol string, side binance.SideType, qty string) *binance.WsApiRequest {
	params := binance.NewOrderTradeParmes(t.binanceApiKey).
		NewOrderRespType(binance.NewOrderRespTypeRESULT).
		TimeInForce(binance.TimeInForceTypeGTC).
		Symbol(symbol).Side(side).
		OrderType(binance.OrderTypeMarket).Quantity(qty).
		Signature(t.binanceSecretKey)

	if TestTrade {
		return binance.NewOrderTradeTest(params)
	}
	return binance.NewOrderTrade(params)
}
