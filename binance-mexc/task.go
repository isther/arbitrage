package binancemexc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
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

	closeRatio       decimal.Decimal
	openStablePrice  decimal.Decimal
	openBinancePrice decimal.Decimal
	openMexcPrice    decimal.Decimal

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
		profitRatio:      decimal.NewFromFloat(ratio).Div(base),
		minRatio:         decimal.NewFromFloat(minRatio).Div(base),
		maxRatio:         decimal.NewFromFloat(maxRatio).Div(base),
	}
}

func (t *Task) run(
	binanceSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent,
	mexcSymbolEventCh chan *mexc.WsBookTickerEvent,
	OrderIDsCh chan OrderIds,
) {
	// Init
	t.mode.Store(0)
	t.isOpen.Store(true)

	var (
		doCh = make(chan struct{}, 100)
	)

	go t.trade(OrderIDsCh, doCh)

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
			logrus.Debug("Stop")
			return
		}
	}
}

func (t *Task) Init() {
	t.mode.Store(0)
	number++
	if config.Config.Number == number {
		truelyPause()
		logrus.Warnf("已完成任务%d次，停止。", number)
	} else {
		pauseCh <- struct{}{}
		time.Sleep(time.Duration(config.Config.WaitDuration) * time.Millisecond)
		unPauseCh <- struct{}{}
	}
}

func (t *Task) trade(
	orderIDsCh chan OrderIds,
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

		var (
			ok       bool
			orderIDs OrderIds
		)
		// Open
		for {
			<-doCh

			// Check
			// 1. 网络延达超过?ms，暂停套利?秒
			// 2. BTC振福超过?万，暂停套利?秒
			if Paused.Load() {
				continue
			}
			t.isOpen.Store(true)
			ok, t.closeRatio, t.openStablePrice, t.openBinancePrice, t.openMexcPrice,
				orderIDs.OpenBinanceID = t.open()
			if ok {
				time.Sleep(1888 * time.Millisecond)
				break
			}
		}
		// Close
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Config.CloseTimeOut)*time.Millisecond)
		defer cancel()

		orderIDs.CloseBinanceID = t.close(ctx)
		orderIDs.mode = t.mode.Load()
		// orderIDsCh <- orderIDs
		t.Init()
	}
}

// 开仓
func (t *Task) open() (bool, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, string) {
	if ok, ratio, stableSymbolPrice, binanceSymbolPrice, mexcSymbolPrice,
		binanceOrderID :=
		t.openMode1(); ok {
		t.isOpen.Store(false)
		logrus.Debug("--------------> Mode2 平仓")
		return true, ratio, stableSymbolPrice, binanceSymbolPrice, mexcSymbolPrice, binanceOrderID
	} else if ok, ratio, stableSymbolPrice, binanceSymbolPrice, mexcSymbolPrice,
		binanceOrderID :=
		t.openMode2(); ok {
		t.isOpen.Store(false)
		logrus.Debug("--------------> Mode1 平仓")
		return true, ratio, stableSymbolPrice, binanceSymbolPrice, mexcSymbolPrice, binanceOrderID
	}

	return false, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ""
}

// 模式1
// （tusd/usdt区的买1价）减去（btc/usdt区卖1价除以btc/tusd区买1价）大于万0.7 小于万1.5
func (t *Task) openMode1() (bool, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, string) {
	// Prepare price
	stableSymbolBidPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestBidPrice)
	mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.AskPrice)
	binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestBidPrice)

	ratioMode1, ok := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, stableSymbolBidPrice)
	if !ok {
		return false, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ""
	}

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
		binanceOrderID := fmt.Sprintf("O1%d", time.Now().UnixNano())
		t.tradeMode1(
			binanceOrderID,
			config.Config.MaxQty,
		)
		stableSymbolAskPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestAskPrice)
		return true, ratioMode1, stableSymbolAskPrice, binanceSymbolBidPrice, mexcSymbolAskPrice, binanceOrderID
	}
	return false, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ""
}

// 模式2
// （1除以tusd/usdt区的卖1价）减去（btc/tusd区卖1价除以btc/usdt区买1价）大于万0.7 小于万1.5
func (t *Task) openMode2() (bool, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, string) {
	// Prepare price
	stableSymbolAskPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestAskPrice)
	binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
	mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

	ratioMode2, ok := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, stableSymbolAskPrice)
	if !ok {
		return false, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ""
	}

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
		binanceOrderID := fmt.Sprintf("O2%d", time.Now().UnixNano())
		t.tradeMode2(
			binanceOrderID,
			config.Config.MaxQty,
		)
		stableSymbolBidPrice, _ := decimal.NewFromString(t.stableCoinSymbolEvent.BestBidPrice)
		return true, ratioMode2, stableSymbolBidPrice, binanceSymbolAskPrice, mexcSymbolBidPrice, binanceOrderID
	}

	return false, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ""
}

// 平仓
func (t *Task) close(
	ctx context.Context,
) string {
	for {
		select {
		case <-ctx.Done():
			return t.foreceClose()
		default:
			switch t.mode.Load() {
			case 1:
				// 做模式2
				ratio := decimal.NewFromFloat(-0.0001).Sub(t.closeRatio).Add(t.profitRatio)
				binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
				mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)
				ratioMode2, ok := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, t.openStablePrice)
				if !ok {
					continue
				}

				if ratioMode2.GreaterThanOrEqual(ratio) {
					logrus.Info(t.ratioLog(ratioMode2, t.openStablePrice, binanceSymbolAskPrice, mexcSymbolBidPrice))
					t.profitLog(binanceSymbolAskPrice, mexcSymbolBidPrice)

					// Trade
					binanceOrderID := fmt.Sprintf("C1%d", time.Now().UnixNano())
					t.tradeMode2(
						binanceOrderID,
						config.Config.MaxQty,
					)
					return binanceOrderID
				}
			case 2:
				// 做模式1
				ratio := decimal.NewFromFloat(-0.0001).Sub(t.closeRatio).Add(t.profitRatio)
				mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.AskPrice)
				binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestBidPrice)
				ratioMode1, ok := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, t.openStablePrice)
				if !ok {
					continue
				}

				if ratioMode1.GreaterThanOrEqual(ratio) {
					logrus.Info(t.ratioLog(ratioMode1, t.openStablePrice, binanceSymbolBidPrice, mexcSymbolAskPrice))
					t.profitLog(binanceSymbolBidPrice, mexcSymbolAskPrice)

					// Trade
					binanceOrderID := fmt.Sprintf("C2%d", time.Now().UnixNano())
					t.tradeMode1(
						binanceOrderID,
						config.Config.MaxQty,
					)
					return binanceOrderID
				}
			}
		}
	}
}

func (t *Task) foreceClose() (binanceOrderID string) {
	binanceOrderID = fmt.Sprintf("FC%d%d", t.mode.Load(), time.Now().UnixNano())
	switch t.mode.Load() {
	case 1:
		// Get Price
		binanceSymbolAskPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
		mexcSymbolBidPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

		// Log
		ratio, _ := t.calculateRatioMode2(binanceSymbolAskPrice, mexcSymbolBidPrice, t.openStablePrice)
		logrus.Info("[强平]",
			t.ratioLog(
				ratio,
				t.openStablePrice,
				binanceSymbolAskPrice,
				mexcSymbolBidPrice,
			),
		)
		t.profitLog(binanceSymbolAskPrice, mexcSymbolBidPrice)

		t.tradeMode2(
			binanceOrderID,
			config.Config.MaxQty,
		)
	case 2:
		// Get Price
		binanceSymbolBidPrice, _ := decimal.NewFromString(t.binanceSymbolEvent.BestAskPrice)
		mexcSymbolAskPrice, _ := decimal.NewFromString(t.mexcSymbolEvent.Data.BidPrice)

		// Log
		ratio, _ := t.calculateRatioMode1(binanceSymbolBidPrice, mexcSymbolAskPrice, t.openStablePrice)
		logrus.Info("[强平]",
			t.ratioLog(
				ratio,
				t.openStablePrice,
				binanceSymbolBidPrice,
				mexcSymbolAskPrice,
			),
		)
		t.profitLog(binanceSymbolBidPrice, mexcSymbolAskPrice)

		t.tradeMode1(
			binanceOrderID,
			config.Config.MaxQty,
		)
	}
	return binanceOrderID
}

func (t *Task) tradeMode1(
	newClientOrderId,
	binanceQty string,
) {
	t.binanceTrade(
		newClientOrderId,
		"BTCUSDT",
		binancesdk.SideTypeBuy,
		binanceQty,
	)
}

func (t *Task) tradeMode2(
	newClientOrderId,
	binanceQty string,
) {

	t.binanceTrade(
		newClientOrderId,
		"BTCUSDT",
		binancesdk.SideTypeSell,
		binanceQty,
	)
}

func (t *Task) calculateRatioMode1(taPrice, tbPrice, stableSymbolPrice decimal.Decimal) (decimal.Decimal, bool) {
	if taPrice.IsZero() || tbPrice.IsZero() || stableSymbolPrice.IsZero() {
		return decimal.Zero, false
	}

	return stableSymbolPrice.
		Sub(
			tbPrice.Div(taPrice),
		), true
}

func (t *Task) calculateRatioMode2(taPrice, tbPrice, stableSymbolPrice decimal.Decimal) (decimal.Decimal, bool) {
	if taPrice.IsZero() || tbPrice.IsZero() || stableSymbolPrice.IsZero() {
		return decimal.Zero, false
	}

	return decimal.NewFromFloat32(1).Div(stableSymbolPrice).
		Sub(
			taPrice.Div(
				tbPrice,
			),
		), true
}

func (t *Task) ratioLog(ratio, stableSymbolPrice, taPrice, tbPrice decimal.Decimal) string {
	var status string
	if t.isOpen.Load() {
		status = "Open"
	} else {
		status = "Close"
	}
	// return fmt.Sprintf(
	// 	"Status: %s [Mode%d] TUSD/USDT: %s BTC/TUSD: %s BTC/USDT: %s Ratio: %s",
	// 	status,
	// 	t.mode.Load(),
	// 	stableSymbolPrice,
	// 	taPrice,
	// 	tbPrice,
	// 	ratio.Mul(decimal.NewFromFloat(10000)).String(),
	// )
	return fmt.Sprintf(
		"Status: %s [Mode%d] BTC/TUSD: %s BTC/USDT: %s Ratio: %s",
		status,
		t.mode.Load(),
		taPrice,
		tbPrice,
		ratio.Mul(decimal.NewFromFloat(10000)).String(),
	)
}

func (t *Task) profitLog(closeBinancePrice, closeMexcPrice decimal.Decimal) {
	var (
		tusdProfit = decimal.Zero
		usdtProfit = decimal.Zero
	)

	switch t.mode.Load() {
	case 1:
		tusdProfit = t.openBinancePrice.Sub(closeBinancePrice)
		usdtProfit = closeMexcPrice.Sub(t.openMexcPrice)
	case 2:
		tusdProfit = closeBinancePrice.Sub(t.openBinancePrice)
		usdtProfit = t.openMexcPrice.Sub(closeMexcPrice)
	default:
		panic("Invalid mode")
	}

	msg := fmt.Sprintf("模式%d \n[开仓]: BTC/TUSD: %s BTC/USDT: %s\n[平仓]: BTC/TUSD: %s BTC/USDT: %s\n[预期盈利] BTC/TUSD: %s BTC/USDT: %s\n[合计预期结果] %s",
		t.mode.Load(),
		t.openBinancePrice.String(), t.openMexcPrice.String(),
		closeBinancePrice.String(), closeMexcPrice.String(),
		tusdProfit.String(), usdtProfit.String(),
		tusdProfit.Add(usdtProfit).String())

	logrus.Infof(msg)

}

func (t *Task) binanceTrade(newClientOrderId, symbol string, side binancesdk.SideType, qty string) {
	quantityDecimal, _ := decimal.NewFromString(qty)
	quantityDecimal = quantityDecimal.Truncate(5).Truncate(8)
	qty = quantityDecimal.String()

	res, err := binancesdk.NewClient(t.binanceApiKey, t.binanceSecretKey).NewCreateOrderService().
		Symbol(symbol).Side(side).Type(binancesdk.OrderTypeMarket).
		Quantity(qty).NewClientOrderID(newClientOrderId).
		Do(context.Background())
	if err != nil {
		logrus.Error(res, err)
	}
}
