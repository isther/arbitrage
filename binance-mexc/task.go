package binancemexc

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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

	MinRatio decimal.Decimal
	MaxRatio decimal.Decimal
	Ratio    decimal.Decimal

	stopCh chan struct{}

	isOpen atomic.Bool
	mode   atomic.Int32
	L      sync.RWMutex
}

func NewArbitrageTask(
	binanceApiKey,
	binanceSecretKey,
	mexcApiKey,
	mexcSecretKey string,
	symbolPairs SymbolPair,
	ratio, minRatio, maxRatio float64) *Task {
	return &Task{
		binanceApiKey:    binanceApiKey,
		binanceSecretKey: binanceSecretKey,
		mexcApiKey:       mexcApiKey,
		mexcSecretKey:    mexcSecretKey,
		symbolPairs:      symbolPairs,
		stopCh:           make(chan struct{}),
		Ratio:            decimal.NewFromFloat(ratio),
		MinRatio:         decimal.NewFromFloat(minRatio),
		MaxRatio:         decimal.NewFromFloat(maxRatio),
	}
}

func (t *Task) run(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	mexcSymbolEventCh chan *mexc.WsBookTickerEvent,
) {

	var (
		ratio             decimal.Decimal
		stableSymbolPrice decimal.Decimal
		trade             = func() {
			stableEvent, binanceEvent := t.stableCoinSymbolEvent, t.binanceSymbolEvent
			mexcEvent := t.mexcSymbolEvent
			if stableEvent == nil || binanceEvent == nil || mexcEvent == nil {
				// log.Println("Get nil event")
				return
			}
			if mexcEvent.Data.AskPrice == "" {
				// log.Println("Get null mexc event")
				return
			}

			switch t.mode.Load() {
			case 0: // Open
				t.isOpen.Store(true)
				ratio, stableSymbolPrice = t.open(binanceWsReqCh, binanceEvent, stableEvent, mexcEvent)
			case 1, 2:
				t.close(binanceWsReqCh, ratio, stableSymbolPrice, binanceEvent, mexcEvent)
			}
		}
		getCtx = func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 1000*time.Second)
		}
		ctx, cancel = getCtx()
	)

	t.mode.Store(0)
	t.isOpen.Store(true)
	for {
		select {
		case binanceSymbolEvent := <-binanceSymbolEventCh:
			go func() {
				switch t.L.TryLock() {
				case true:
					{
						defer t.L.Unlock()
						t.binanceSymbolEvent = binanceSymbolEvent
						trade()
					}
				}
			}()
		case stableCoinSymbolEvent := <-stableCoinSymbolEventCh:
			go func() {
				switch t.L.TryLock() {
				case true:
					{
						defer t.L.Unlock()
						t.stableCoinSymbolEvent = stableCoinSymbolEvent
						trade()
					}
				}
			}()
		case mexcSymbolEvent := <-mexcSymbolEventCh:
			go func() {
				switch t.L.TryLock() {
				case true:
					{
						defer t.L.Unlock()
						t.mexcSymbolEvent = mexcSymbolEvent
						trade()
					}
				}
			}()
		case <-ctx.Done():
			cancel()
			ctx, cancel = getCtx()
			switch t.mode.Load() {
			case 1:
				go func() {
					t.L.Lock()
					defer t.L.Unlock()

					mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)
					t.tradeMode2(binanceWsReqCh, "0.0004", mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(), "0.0004")
				}()
			case 2:
				go func() {
					t.L.Lock()
					defer t.L.Unlock()

					mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.AskPrice)
					t.tradeMode1(binanceWsReqCh, "0.0004", mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(), "0.0004")
				}()
			}
			t.mode.Store(0)
		case <-t.stopCh:
			log.Println("Stop")
			return
		}
	}
}

// 开仓
func (t *Task) open(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEvent *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) (decimal.Decimal, decimal.Decimal) {

	if ok, ratio, stableSymbolPrice := t.openMode1(binanceWsReqCh, binanceSymbolEvent, stableCoinSymbolEvent, mexcSymbolEvent); ok {
		t.isOpen.Store(false)
		log.Println("--------------> Mode2 平仓")
		return ratio, stableSymbolPrice
	} else if ok, ratio, stableSymbolPrice := t.openMode2(binanceWsReqCh, binanceSymbolEvent, stableCoinSymbolEvent, mexcSymbolEvent); ok {
		t.isOpen.Store(false)
		log.Println("--------------> Mode1 平仓")
		return ratio, stableSymbolPrice
	}

	return decimal.Zero, decimal.Zero
}

// 模式1
// （tusd/usdt区的买1价）减去（btc/usdt区卖1价除以btc/tusd区买1价）大于万0.7 小于万1.5
func (t *Task) openMode1(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEvent,
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) (bool, decimal.Decimal, decimal.Decimal) {
	// Prepare price
	stableSymbolBidPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestBidPrice)
	mexcSymbolAskPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.AskPrice)
	binanceSymbolBidPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestBidPrice)

	ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolBidPrice)
	if ratioMode1.GreaterThanOrEqual(t.MinRatio) && ratioMode1.LessThanOrEqual(t.MaxRatio) {
		t.mode.Store(1)
		t.ratioLog(ratioMode1, stableSymbolBidPrice, binanceSymbolBidPrice, mexcSymbolAskPrice)
		// 币安把BTC卖出为TUSD、抹茶把USDT买入为BTC；
		// stableSymbolBidQty, _ := decimal.NewFromString(stableEvent.BestBidQty)   // TUSDUSDT
		// mexcSymbolAskQty, _ := decimal.NewFromString(mexcEvent.Data.AskQty)      // BTCUSDT
		// binanceSymbolBidQty, _ := decimal.NewFromString(binanceEvent.BestBidQty) //BTCTUSD

		// stabldToAQty := stableSymbolBidQty
		// binanceToAQty := binanceSymbolBidQty.Mul(binanceSymbolBidPrice)
		// mexcToAQty := mexcSymbolAskQty.Mul(mexcSymbolAskPrice).Div(stableSymbolBidPrice)

		// Quantity
		// aQty := decimal.Min(stabldToAQty, binanceToAQty, mexcToAQty, decimal.NewFromFloat(11.0))
		// aQty = decimal.Max(aQty, decimal.NewFromFloat(11.0))

		// Trade binance
		if TestTrade {

		} else {
			t.tradeMode1(binanceWsReqCh, "0.0004", mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(), "0.0004")
		}
		stableSymbolAskPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestAskPrice)
		return true, ratioMode1, stableSymbolAskPrice
	}
	return false, decimal.Zero, decimal.Zero
}

// 模式2
// （1除以tusd/usdt区的卖1价）减去（btc/tusd区卖1价除以btc/usdt区买1价）大于万0.7 小于万1.5
func (t *Task) openMode2(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEvent,
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) (bool, decimal.Decimal, decimal.Decimal) {
	// Prepare price
	stableSymbolAskPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestAskPrice)
	binanceSymbolAskPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestAskPrice)
	mexcSymbolBidPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.BidPrice)

	ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolAskPrice)
	if ratioMode2.GreaterThanOrEqual(t.MinRatio) && ratioMode2.LessThanOrEqual(t.MaxRatio) {
		t.mode.Store(2)
		t.ratioLog(ratioMode2, stableSymbolAskPrice, binanceSymbolAskPrice, mexcSymbolBidPrice)
		// 币安把TUSD买入为BTC、抹茶把BTC卖出为USDT；
		// stableSymbolAskQty, _ := decimal.NewFromString(stableEvent.BestAskQty)   // TUSDUSDT
		// binanceSymbolAskQty, _ := decimal.NewFromString(binanceEvent.BestAskQty) // BTCTUSD
		// mexcSymbolBidQty, _ := decimal.NewFromString(mexcEvent.Data.BidQty)      // BTCUSDT

		// stabldToAQty := stableSymbolAskQty
		// binanceToAQty := binanceSymbolAskQty.Mul(binanceSymbolAskPrice)
		// mexcToAQty := mexcSymbolBidQty.Mul(mexcSymbolBidPrice).Div(stableSymbolAskPrice)

		// Quantity
		// aQty := decimal.Min(stabldToAQty, binanceToAQty, mexcToAQty, decimal.NewFromFloat(11.0))
		// aQty = decimal.Max(aQty, decimal.NewFromFloat(11.0))

		// Trade binance
		if TestTrade {

		} else {
			t.tradeMode2(binanceWsReqCh, "0.0004", mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(), "0.0004")
		}
		stableSymbolBidPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestBidPrice)
		return true, ratioMode2, stableSymbolBidPrice
	}

	return false, decimal.Zero, decimal.Zero
}

// 平仓
func (t *Task) close(
	binanceWsReqCh chan *binance.WsApiRequest,
	ratio,
	stableSymbolPrice decimal.Decimal,
	binanceSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) {
	switch t.mode.Load() {
	case 1:
		// 做模式2
		ratio = decimal.NewFromFloat(-0.0001).Sub(ratio).Add(t.Ratio)
		binanceSymbolAskPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestAskPrice)
		mexcSymbolBidPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.BidPrice)

		if ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolPrice); ratioMode2.GreaterThanOrEqual(ratio) {
			t.ratioLog(ratioMode2, stableSymbolPrice, binanceSymbolAskPrice, mexcSymbolBidPrice)

			// Trade
			t.tradeMode2(binanceWsReqCh, "0.0004", mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(), "0.0004")
			t.mode.Store(0)
			time.Sleep(1 * time.Second)
			os.Exit(-1)
		}
	case 2:
		// 做模式1
		ratio = decimal.NewFromFloat(-0.0001).Sub(ratio).Add(t.Ratio)
		mexcSymbolAskPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.AskPrice)
		binanceSymbolBidPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestBidPrice)

		if ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolPrice); ratioMode1.GreaterThanOrEqual(ratio) {
			t.ratioLog(ratioMode1, stableSymbolPrice, binanceSymbolBidPrice, mexcSymbolAskPrice)

			// Trade
			t.tradeMode1(binanceWsReqCh, "0.0004", mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(), "0.0004")
			t.mode.Store(0)
			time.Sleep(1 * time.Second)
			os.Exit(-1)
		}
	}
}

func (t *Task) tradeMode1(binanceWsReqCh chan *binance.WsApiRequest, binanceQty, mexcPrice, mexcQty string) {
	if t.isOpen.Load() {
		log.Println("Open trade mode1 start")
		defer log.Println("Open trade mode1 end")
	} else {
		log.Println("Close trade mode1 start")
		defer log.Println("Close trade mode1 end")
	}

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		binanceWsReqCh <- t.getOrderBinanceTrade(
			t.symbolPairs.BinanceSymbol,
			binance.SideTypeSell,
			binanceQty,
		)
	}()

	// Trade mexc
	go func() {
		wg.Add(1)
		defer wg.Done()

		if res, err := mexc.MexcBTCBuy(
			config.Config.MexcCookie,
			mexcPrice,
			mexcQty,
		); err != nil {
			log.Println("mexc trade error", err)
		} else {
			log.Println("mexc trade", res)
		}
	}()

	wg.Wait()
}

func (t *Task) tradeMode2(binanceWsReqCh chan *binance.WsApiRequest, binanceQty, mexcPrice, mexcQty string) {
	if t.isOpen.Load() {
		log.Println("Open trade mode2 start")
		defer log.Println("Open trade mode2 end")
	} else {
		log.Println("Close with trade mode2 start")
		defer log.Println("Close with 2rade mode2 end")
	}
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		binanceWsReqCh <- t.getOrderBinanceTrade(
			t.symbolPairs.BinanceSymbol,
			binance.SideTypeBuy,
			binanceQty,
		)
	}()

	// Trade mexc
	go func() {
		wg.Add(1)
		defer wg.Done()

		if res, err := mexc.MexcBTCSell(
			config.Config.MexcCookie,
			mexcPrice,
			mexcQty,
		); err != nil {
			log.Println("mexc trade error", err)
		} else {
			log.Println("mexc trade", res)
		}
	}()

	wg.Wait()
}

func (t *Task) calculateRatioMode1(taPrice, tbPrice, stableSymbolPrice decimal.Decimal) decimal.Decimal {
	return stableSymbolPrice.
		Sub(
			tbPrice.Div(taPrice),
		)
}

func (t *Task) calculateRatioMode2(taPrice, tbPrice, stableSymbolPrice decimal.Decimal) decimal.Decimal {
	return decimal.NewFromFloat32(1).Div(stableSymbolPrice).
		Sub(
			taPrice.Div(
				tbPrice,
			),
		)
}

func (t *Task) ratioLog(ratio, stableSymbolPrice, taPrice, tbPrice decimal.Decimal) {
	var status string
	if t.isOpen.Load() {
		status = "Open"
	} else {
		status = "Close"
	}
	log.Println(
		fmt.Sprintf(
			"Status: %s [Mode%d] TUSD/USDT: %s BTC/TUSD: %s BTC/USDT: %s Ratio: %s",
			status,
			t.mode.Load(),
			stableSymbolPrice,
			taPrice,
			tbPrice,
			ratio.Mul(decimal.NewFromFloat(10000)).String(),
		),
	)

}

func (t *Task) getOrderBinanceTrade(symbol string, side binance.SideType, qty string) *binance.WsApiRequest {
	params := binance.NewOrderTradeParmes(t.binanceApiKey).
		NewOrderRespType(binance.NewOrderRespTypeRESULT).
		Symbol(symbol).Side(side).
		OrderType(binance.OrderTypeMarket).Quantity(qty)

	if runtime.GOOS != "linux" {
		params.TimeInForce(binance.TimeInForceTypeGTC)
	}

	return binance.NewOrderTrade(params.
		Signature(t.binanceSecretKey))
}
