package binancemexc

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var (
	number int64 = 0
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

	closeStablePrice decimal.Decimal
	closeRatio       decimal.Decimal

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
	openMexcOrderIdCh chan string,
	closeMexcOrderIdCh chan string,
	binanceSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	mexcSymbolEventCh chan *mexc.WsBookTickerEvent,
) {
	// Init
	t.mode.Store(0)
	t.isOpen.Store(true)

	var (
		doCh = make(chan struct{}, 100)
	)

	go t.trade(binanceWsReqCh, openMexcOrderIdCh, closeMexcOrderIdCh, doCh)

	for {
		select {
		case t.binanceSymbolEvent = <-binanceSymbolEventCh:
			// t.binanceSymbolEvent = binanceSymbolEvent
			go func() { doCh <- struct{}{} }()
			time.Sleep(1 * time.Millisecond)
		case t.stableCoinSymbolEvent = <-stableCoinSymbolEventCh:
			// t.stableCoinSymbolEvent = stableCoinSymbolEvent
			go func() { doCh <- struct{}{} }()
			time.Sleep(1 * time.Millisecond)
		case t.mexcSymbolEvent = <-mexcSymbolEventCh:
			// t.mexcSymbolEvent = mexcSymbolEvent
			go func() { doCh <- struct{}{} }()
			time.Sleep(1 * time.Millisecond)
		case <-t.stopCh:
			logrus.Info("Stop")
			return
		}
	}
}

func (t *Task) Init() {
	time.Sleep(5 * time.Second)
	t.mode.Store(0)

	number++
	if config.Config.Number == number {
		os.Exit(-1)
	}
}

func (t *Task) trade(
	binanceWsReqCh chan *binance.WsApiRequest,
	openMexcOrderIdCh chan string,
	closeMexcOrderIdCh chan string,
	doCh chan struct{},
) {

	for {
		<-doCh
		// stableEvent, binanceEvent := t.stableCoinSymbolEvent, t.binanceSymbolEvent
		// mexcEvent := t.mexcSymbolEvent
		if t.stableCoinSymbolEvent == nil || t.binanceSymbolEvent == nil || t.mexcSymbolEvent == nil {
			logrus.Debug("Get nil event")
			continue
		}
		if t.mexcSymbolEvent.Data.AskPrice == "" {
			logrus.Debug("Get null mexc event")
			continue
		}

		logrus.Infof("Start task: MinRatio: %s, MaxRatio: %s, ProfitRatio: %s, CloseTimeOut: %v\n",
			t.minRatio,
			t.maxRatio,
			t.profitRatio,
			config.Config.CloseTimeOut,
		)

		switch t.mode.Load() {
		case 0:
			var (
				ok bool
			)
			for {
				<-doCh
				t.isOpen.Store(true)
				ok, t.closeRatio, t.closeStablePrice = t.open(binanceWsReqCh, openMexcOrderIdCh)
				if ok {
					break
				}
			}
		case 1, 2:
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Config.CloseTimeOut)*time.Millisecond)
			defer cancel()

			t.close(binanceWsReqCh, closeMexcOrderIdCh, ctx)
			t.Init()
		}
	}
}

// 开仓
func (t *Task) open(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
) (bool, decimal.Decimal, decimal.Decimal) {

	if ok, ratio, stableSymbolPrice := t.openMode1(
		binanceWsReqCh,
		mexcOrderIdCh,
	); ok {
		t.isOpen.Store(false)
		logrus.Info("--------------> Mode2 平仓")
		return true, ratio, stableSymbolPrice
	} else if ok, ratio, stableSymbolPrice := t.openMode2(
		binanceWsReqCh,
		mexcOrderIdCh,
	); ok {
		t.isOpen.Store(false)
		logrus.Info("--------------> Mode1 平仓")
		return true, ratio, stableSymbolPrice
	}

	return false, decimal.Zero, decimal.Zero
}

// 模式1
// （tusd/usdt区的买1价）减去（btc/usdt区卖1价除以btc/tusd区买1价）大于万0.7 小于万1.5
func (t *Task) openMode1(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
) (bool, decimal.Decimal, decimal.Decimal) {
	// Prepare price
	stableSymbolBidPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestBidPrice)
	mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.AskPrice)
	binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestBidPrice)

	ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolBidPrice)
	if ratioMode1.GreaterThanOrEqual(t.minRatio) && ratioMode1.LessThanOrEqual(t.maxRatio) {
		t.mode.Store(1)
		logrus.Info(t.ratioLog(ratioMode1, stableSymbolBidPrice, binanceSymbolBidPrice, mexcSymbolAskPrice))
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
			orderID := fmt.Sprintf("O1%d", time.Now().UnixNano())
			t.tradeMode1(
				binanceWsReqCh,
				mexcOrderIdCh,
				orderID,
				"0.0004",
				mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(),
				"0.0004",
			)
		}
		stableSymbolAskPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestAskPrice)
		return true, ratioMode1, stableSymbolAskPrice
	}
	return false, decimal.Zero, decimal.Zero
}

// 模式2
// （1除以tusd/usdt区的卖1价）减去（btc/tusd区卖1价除以btc/usdt区买1价）大于万0.7 小于万1.5
func (t *Task) openMode2(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
) (bool, decimal.Decimal, decimal.Decimal) {
	// Prepare price
	stableSymbolAskPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestAskPrice)
	binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
	mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

	ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolAskPrice)
	if ratioMode2.GreaterThanOrEqual(t.minRatio) && ratioMode2.LessThanOrEqual(t.maxRatio) {
		t.mode.Store(2)
		logrus.Info(t.ratioLog(ratioMode2, stableSymbolAskPrice, binanceSymbolAskPrice, mexcSymbolBidPrice))
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
			orderID := fmt.Sprintf("O2%d", time.Now().UnixNano())
			t.tradeMode2(
				binanceWsReqCh,
				mexcOrderIdCh,
				orderID,
				"0.0004",
				mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(),
				"0.0004",
			)
		}
		stableSymbolBidPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestBidPrice)
		return true, ratioMode2, stableSymbolBidPrice
	}

	return false, decimal.Zero, decimal.Zero
}

// 平仓
func (t *Task) close(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
	ctx context.Context,
) {
	for {
		select {
		case <-ctx.Done():
			t.foreceClose(binanceWsReqCh, mexcOrderIdCh)
			return
		default:
			switch t.mode.Load() {
			case 1:
				// 做模式2
				ratio := decimal.NewFromFloat(-0.0001).Sub(t.closeRatio).Add(t.profitRatio)
				binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
				mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

				if ratioMode2 := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, t.closeStablePrice); ratioMode2.GreaterThanOrEqual(ratio) {
					logrus.Info(t.ratioLog(ratioMode2, t.closeStablePrice, binanceSymbolAskPrice, mexcSymbolBidPrice))

					// Trade
					orderID := fmt.Sprintf("C1%d", time.Now().UnixNano())
					t.tradeMode2(
						binanceWsReqCh,
						mexcOrderIdCh,
						orderID,
						"0.0004",
						mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(),
						"0.0004",
					)
					return
				}
			case 2:
				// 做模式1
				ratio := decimal.NewFromFloat(-0.0001).Sub(t.closeRatio).Add(t.profitRatio)
				mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.AskPrice)
				binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestBidPrice)

				if ratioMode1 := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, t.closeStablePrice); ratioMode1.GreaterThanOrEqual(ratio) {
					logrus.Info(t.ratioLog(ratioMode1, t.closeStablePrice, binanceSymbolBidPrice, mexcSymbolAskPrice))

					// Trade
					orderID := fmt.Sprintf("C2%d", time.Now().UnixNano())
					t.tradeMode1(
						binanceWsReqCh,
						mexcOrderIdCh,
						orderID,
						"0.0004",
						mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(),
						"0.0004",
					)
					return
				}
			}
		}
	}
}

func (t *Task) foreceClose(
	binanceWsReqCh chan *binance.WsApiRequest,
	closeMexcOrderIdCh chan string,
) {
	orderID := fmt.Sprintf("FC%d%d", t.mode.Load(), time.Now().UnixNano())
	switch t.mode.Load() {
	case 1:
		// Get Price
		binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
		mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

		// Log
		logrus.Info("[强平]",
			t.ratioLog(
				t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, t.closeStablePrice),
				t.closeStablePrice,
				binanceSymbolAskPrice,
				mexcSymbolBidPrice,
			),
		)

		t.tradeMode2(
			binanceWsReqCh,
			closeMexcOrderIdCh,
			orderID,
			"0.0004",
			mexcSymbolBidPrice.Mul(decimal.NewFromFloat(0.99)).String(),
			"0.0004",
		)
	case 2:
		// Get Price
		binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
		mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

		// Log
		logrus.Info("[强平]",
			t.ratioLog(
				t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, t.closeStablePrice),
				t.closeStablePrice,
				binanceSymbolBidPrice,
				mexcSymbolAskPrice,
			),
		)

		t.tradeMode1(
			binanceWsReqCh,
			closeMexcOrderIdCh,
			orderID,
			"0.0004",
			mexcSymbolAskPrice.Mul(decimal.NewFromFloat(1.01)).String(),
			"0.0004",
		)
	}
}

func (t *Task) tradeMode1(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
	newClientOrderId,
	binanceQty,
	mexcPrice,
	mexcQty string,
) {
	if t.isOpen.Load() {
		logrus.Info("Open trade mode1 start")
		defer logrus.Info("Open trade mode1 end")
	} else {
		logrus.Info("Close trade mode1 start")
		defer logrus.Info("Close trade mode1 end")
	}

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		binanceWsReqCh <- t.getOrderBinanceTrade(
			newClientOrderId,
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
			logrus.Error("mexc trade error", err, " res: ", res)
		} else {
			logrus.Println(res)
			mexcOrderIdCh <- res.Data
		}
	}()

	wg.Wait()
}

func (t *Task) tradeMode2(
	binanceWsReqCh chan *binance.WsApiRequest,
	mexcOrderIdCh chan string,
	newClientOrderId,
	binanceQty,
	mexcPrice,
	mexcQty string,
) {
	if t.isOpen.Load() {
		logrus.Info("Open trade mode2 start")
		defer logrus.Info("Open trade mode2 end")
	} else {
		logrus.Info("Close with trade mode2 start")
		defer logrus.Info("Close with trade mode2 end")
	}

	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()

		binanceWsReqCh <- t.getOrderBinanceTrade(
			newClientOrderId,
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
			logrus.Error("mexc trade error", err, " res: ", res)
		} else {
			logrus.Info(res)
			mexcOrderIdCh <- res.Data
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

func (t *Task) getOrderBinanceTrade(newClientOrderId, symbol string, side binance.SideType, qty string) *binance.WsApiRequest {
	params := binance.NewOrderTradeParmes(t.binanceApiKey).
		NewOrderRespType(binance.NewOrderRespTypeRESULT).
		Symbol(symbol).Side(side).
		OrderType(binance.OrderTypeMarket).Quantity(qty).
		NewClientOrderID(newClientOrderId)

	// if runtime.GOOS != "linux" {
	// params.TimeInForce(binance.TimeInForceTypeGTC)
	// }

	return binance.NewOrderTrade(params.
		Signature(t.binanceSecretKey))
}
