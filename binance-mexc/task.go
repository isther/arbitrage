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

const base = 10000

type Task struct {
	binanceApiKey    string
	binanceSecretKey string

	mexcApiKey    string
	mexcSecretKey string

	symbolPairs SymbolPair

	binanceSymbolEvent    *binancesdk.WsBookTickerEvent
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent
	mexcSymbolEvent       *mexc.WsBookTickerEvent

	minRatio    decimal.Decimal
	maxRatio    decimal.Decimal
	profitRatio decimal.Decimal

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
		profitRatio:      decimal.NewFromFloat(ratio).Div(decimal.NewFromInt(base)),
		minRatio:         decimal.NewFromFloat(minRatio).Div(decimal.NewFromInt(base)),
		maxRatio:         decimal.NewFromFloat(maxRatio).Div(decimal.NewFromInt(base)),
	}
}

func (t *Task) run(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	mexcSymbolEventCh chan *mexc.WsBookTickerEvent,
) {

	t.mode.Store(0)
	t.isOpen.Store(true)
	go t.trade(binanceWsReqCh)

	for {
		select {
		case binanceSymbolEvent := <-binanceSymbolEventCh:
			func() {
				t.L.Lock()
				defer t.L.Unlock()
				t.binanceSymbolEvent = binanceSymbolEvent
			}()
		case stableCoinSymbolEvent := <-stableCoinSymbolEventCh:
			func() {
				t.L.Lock()
				defer t.L.Unlock()
				t.stableCoinSymbolEvent = stableCoinSymbolEvent
			}()
		case mexcSymbolEvent := <-mexcSymbolEventCh:
			func() {
				t.L.Lock()
				defer t.L.Unlock()
				t.mexcSymbolEvent = mexcSymbolEvent
			}()
		case <-t.stopCh:
			log.Println("Stop")
			return
		}
	}
}

func (t *Task) Init() {
	time.Sleep(1 * time.Second)
	t.mode.Store(0)
	os.Exit(-1)
}

func (t *Task) trade(
	binanceWsReqCh chan *binance.WsApiRequest,
) {
	log.Printf("Start task: MinRatio: %s, MaxRatio: %s, ProfitRatio: %s, CloseTimeOut: %v\n",
		t.minRatio,
		t.maxRatio,
		t.profitRatio,
		config.Config.CloseTimeOut,
	)
	var (
		ok                bool
		ratio             decimal.Decimal
		stableSymbolPrice decimal.Decimal
		ctx               context.Context
		cancel            context.CancelFunc
		getCtx            = func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Duration(config.Config.CloseTimeOut)*time.Millisecond)
		}
	)
	for {
		stableEvent, binanceEvent := t.stableCoinSymbolEvent, t.binanceSymbolEvent
		mexcEvent := t.mexcSymbolEvent
		if stableEvent == nil || binanceEvent == nil || mexcEvent == nil {
			// log.Println("Get nil event")
			continue
		}
		if mexcEvent.Data.AskPrice == "" {
			// log.Println("Get null mexc event")
			continue
		}

		for {
			t.isOpen.Store(true)
			ok, ratio, stableSymbolPrice = t.open(binanceWsReqCh, binanceEvent, stableEvent, mexcEvent)
			if ok {
				ctx, cancel = getCtx()
				break
			}
		}

		select {
		case <-t.close(binanceWsReqCh, ratio, stableSymbolPrice, binanceEvent, mexcEvent):
			t.Init()
		case <-ctx.Done():
			// 超时开始强平
			switch t.mode.Load() {
			case 1:
				func() {
					t.L.Lock()
					defer t.L.Unlock()

					// Get Price
					stableSymbolPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestAskPrice)
					binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
					mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

					// Log
					ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolPrice)
					log.Println("[强平]", t.ratioLog(ratioMode2, stableSymbolPrice, binanceSymbolAskPrice, mexcSymbolBidPrice))

					t.tradeMode2(binanceWsReqCh, "0.0004", mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(), "0.0004")

					t.Init()
				}()
			case 2:
				func() {
					t.L.Lock()
					defer t.L.Unlock()

					// Get Price
					stableSymbolPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestBidPrice)
					binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
					mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

					// Log
					ratioMode2 := t.calculateRatioMode2(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolPrice)
					log.Println("[强平]", t.ratioLog(ratioMode2, stableSymbolPrice, binanceSymbolBidPrice, mexcSymbolAskPrice))

					t.tradeMode1(binanceWsReqCh, "0.0004", mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(), "0.0004")

					t.Init()
				}()
			}
			cancel()
			ctx, cancel = getCtx()
		}

	}
}

// 开仓
func (t *Task) open(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEvent *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) (bool, decimal.Decimal, decimal.Decimal) {

	if ok, ratio, stableSymbolPrice := t.openMode1(binanceWsReqCh, binanceSymbolEvent, stableCoinSymbolEvent, mexcSymbolEvent); ok {
		t.isOpen.Store(false)
		log.Println("--------------> Mode2 平仓")
		return true, ratio, stableSymbolPrice
	} else if ok, ratio, stableSymbolPrice := t.openMode2(binanceWsReqCh, binanceSymbolEvent, stableCoinSymbolEvent, mexcSymbolEvent); ok {
		t.isOpen.Store(false)
		log.Println("--------------> Mode1 平仓")
		return true, ratio, stableSymbolPrice
	}

	return false, decimal.Zero, decimal.Zero
}

// 模式1
// （tusd/usdt区的买1价）减去（btc/usdt区卖1价除以btc/tusd区买1价）大于万0.7 小于万1.5
func (t *Task) openMode1(
	binanceWsReqCh chan *binance.WsApiRequest,
	binanceSymbolEvent,
	stableCoinSymbolEvent *binancesdk.WsBookTickerEvent,
	mexcSymbolEvent *mexc.WsBookTickerEvent,
) (bool, decimal.Decimal, decimal.Decimal) {
	t.L.Lock()
	defer t.L.Unlock()

	// Prepare price
	stableSymbolBidPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestBidPrice)
	mexcSymbolAskPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.AskPrice)
	binanceSymbolBidPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestBidPrice)

	ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolBidPrice)
	if ratioMode1.GreaterThanOrEqual(t.minRatio) && ratioMode1.LessThanOrEqual(t.maxRatio) {
		t.mode.Store(1)
		log.Println(t.ratioLog(ratioMode1, stableSymbolBidPrice, binanceSymbolBidPrice, mexcSymbolAskPrice))
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
	t.L.Lock()
	defer t.L.Unlock()

	// Prepare price
	stableSymbolAskPrice, _ := decimal.NewFromString(stableCoinSymbolEvent.BestAskPrice)
	binanceSymbolAskPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestAskPrice)
	mexcSymbolBidPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.BidPrice)

	ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolAskPrice)
	if ratioMode2.GreaterThanOrEqual(t.minRatio) && ratioMode2.LessThanOrEqual(t.maxRatio) {
		t.mode.Store(2)
		log.Println(t.ratioLog(ratioMode2, stableSymbolAskPrice, binanceSymbolAskPrice, mexcSymbolBidPrice))
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
) chan struct{} {
	var ch = make(chan struct{})
	go func() {
		for {
			t.L.Lock()
			switch t.mode.Load() {
			case 1:
				// 做模式2
				ratio = decimal.NewFromFloat(-0.0001).Sub(ratio).Add(t.profitRatio)
				binanceSymbolAskPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestAskPrice)
				mexcSymbolBidPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.BidPrice)

				if ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolPrice); ratioMode2.GreaterThanOrEqual(ratio) {
					log.Println(t.ratioLog(ratioMode2, stableSymbolPrice, binanceSymbolAskPrice, mexcSymbolBidPrice))

					// Trade
					t.tradeMode2(binanceWsReqCh, "0.0004", mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(), "0.0004")
					ch <- struct{}{}
					return
				}
			case 2:
				// 做模式1
				ratio = decimal.NewFromFloat(-0.0001).Sub(ratio).Add(t.profitRatio)
				mexcSymbolAskPrice, _ := decimal.NewFromString(mexcSymbolEvent.Data.AskPrice)
				binanceSymbolBidPrice, _ := decimal.NewFromString(binanceSymbolEvent.BestBidPrice)

				if ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolPrice); ratioMode1.GreaterThanOrEqual(ratio) {
					log.Println(t.ratioLog(ratioMode1, stableSymbolPrice, binanceSymbolBidPrice, mexcSymbolAskPrice))

					// Trade
					t.tradeMode1(binanceWsReqCh, "0.0004", mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(), "0.0004")
					ch <- struct{}{}
					return
				}
			}
			t.L.Unlock()
		}
	}()

	return ch
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

func (t *Task) ratioLog(ratio, stableSymbolPrice, taPrice, tbPrice decimal.Decimal) string {
	var status string
	if t.isOpen.Load() {
		status = "Open"
	} else {
		status = "Close"
	}
	return fmt.Sprintf(
		"Status: %s [Mode%d] TUSD/USDT: %s BTC/TUSD: %s BTC/USDT: %s Ratio: %s",
		status,
		t.mode.Load(),
		stableSymbolPrice,
		taPrice,
		tbPrice,
		ratio.Mul(decimal.NewFromFloat(10000)).String(),
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
