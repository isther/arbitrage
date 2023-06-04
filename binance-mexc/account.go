package binancemexc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/mexc"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type Account struct {
	stableSymbolAsset  Asset
	binanceSymbolAsset Asset
	mexcSymbolAsset    Asset

	OrderIDsCh chan OrderIds

	binanceOrders   map[string]Order
	mexcOrders      map[string]Order
	binanceOrdersCh chan Order
	mexcOrdersCh    chan Order
	L               sync.RWMutex
}

type Asset struct {
	Symbol string
	Qty    decimal.Decimal
	Free   decimal.Decimal
	Locked decimal.Decimal
}

type OrderIds struct {
	mode           int32
	OpenBinanceID  string
	CloseBinanceID string
	OpenMexcID     string
	CloseMexcID    string
}

type Order struct {
	ID    string
	Price decimal.Decimal
	Qty   decimal.Decimal
}

func NewAccount(symbolPair SymbolPair) *Account {
	return &Account{
		stableSymbolAsset:  Asset{Symbol: symbolPair.StableCoinSymbol},
		binanceSymbolAsset: Asset{Symbol: symbolPair.BinanceSymbol},
		mexcSymbolAsset:    Asset{Symbol: symbolPair.MexcSymbol},
		OrderIDsCh:         make(chan OrderIds),
		binanceOrders:      make(map[string]Order),
		mexcOrders:         make(map[string]Order),
		binanceOrdersCh:    make(chan Order),
		mexcOrdersCh:       make(chan Order),
	}
}

func (a *Account) Start() {
	go func() {
		for {
			orderIDs := <-a.OrderIDsCh
			a.profitLog(orderIDs)
		}
	}()

	go func() {
		for {
			order := <-a.binanceOrdersCh
			a.L.Lock()
			a.binanceOrders[order.ID] = order
			a.L.Unlock()

			// logrus.Error(order.ID)
		}
	}()

	go func() {
		for {
			order := <-a.mexcOrdersCh
			a.L.Lock()
			a.mexcOrders[order.ID] = order
			a.L.Unlock()

			// logrus.Error(order.ID)
		}
	}()

	var (
		started            atomic.Bool
		startBinanceWsDone = make(chan struct{})
		startMexcWsDone    = make(chan struct{})
	)
	started.Store(false)

	go func() {
		var (
			restartCh        = make(chan struct{})
			binanceListenKey = binance.CreateListenKey()
		)
		defer binance.CloseListenKey(binanceListenKey)

		go func() {
			for {
				time.Sleep(25 * time.Minute)
				binance.KeepListenKey(binanceListenKey)
			}
		}()

		for {
			_, _ = binance.StartWsUserData(
				binanceListenKey,
				func(event *binancesdk.WsUserDataEvent) {
					switch event.Event {
					case binancesdk.UserDataEventTypeOutboundAccountPosition:
						// a.accountUpdate(event.AccountUpdate)
					case binancesdk.UserDataEventTypeBalanceUpdate:
					case binancesdk.UserDataEventTypeExecutionReport:
						a.orderUpdate(event.OrderUpdate)
					}

				},
				func(err error) {
					if err != nil {
						logrus.Error(err)
						restartCh <- struct{}{}
					}
				},
			)

			if !started.Load() {
				startBinanceWsDone <- struct{}{}
			}

			<-restartCh
		}
	}()

	go func() {
		var (
			restartCh     = make(chan struct{})
			mexcListenKey string
			err           error
			client        = newMexcClient()
		)

		for {
			mexcListenKey, err = client.NewStartUserStreamService().Do(context.Background())
			if err != nil {
				logrus.Warn("get mexc listen key error: ", err)
			} else {
				break
			}
		}
		defer client.NewCloseUserStreamService().ListenKey(mexcListenKey).Do(context.Background())

		go func() {
			time.Sleep(25 * time.Minute)
			client.NewKeepaliveUserStreamService().ListenKey(mexcListenKey).Do(context.Background())
		}()

		for {
			_, _ = mexc.StartWsDealsInfoServer(
				mexcListenKey,
				func(event *mexc.WsPrivateDealsEvent) {
					// logrus.WithFields(logrus.Fields{"server": "mexc account"}).Infof("%+v", event)
					if strings.TrimSpace(event.Price) == "" {
						return
					}

					a.mexcOrdersCh <- Order{
						ID:    event.DealsData.OrderId,
						Price: stringToDecimal(event.Price),
						Qty:   stringToDecimal(event.Qty),
					}
				},
				func(err error) {
					if err != nil {
						logrus.WithFields(logrus.Fields{"server": "mexc account"}).Error(err)
						time.Sleep(time.Duration(mexc.ReconnectMexcAccountInfoSleepDuration) * time.Millisecond)
						restartCh <- struct{}{}
					}
				},
			)

			if !started.Load() {
				startMexcWsDone <- struct{}{}
			}

			<-restartCh
		}
	}()

	<-startBinanceWsDone
	<-startMexcWsDone
	started.Store(true)
	logrus.WithFields(logrus.Fields{"server": "Account"}).Debug("Start account")
}

func (a *Account) accountUpdate(accountUpdates binancesdk.WsAccountUpdateList) {
	for _, accountUpdate := range accountUpdates.WsAccountUpdates {
		if accountUpdate.Asset == a.stableSymbolAsset.Symbol {
		}
		if accountUpdate.Asset == a.binanceSymbolAsset.Symbol {
		}
	}

	logrus.Infof("账户余额更新: %v: %v %v\n %v: %v %v\n %v: %v %v\n",
		a.stableSymbolAsset.Symbol, a.stableSymbolAsset.Free, a.stableSymbolAsset.Locked,
		a.binanceSymbolAsset.Symbol, a.binanceSymbolAsset.Free, a.binanceSymbolAsset.Locked,
		a.mexcSymbolAsset.Symbol, a.mexcSymbolAsset.Free, a.mexcSymbolAsset.Locked)
}

func (a *Account) orderUpdate(orderUpdate binancesdk.WsOrderUpdate) {
	switch orderUpdate.Status {
	case "NEW":
		// logrus.Infof("[CREATE]: ID: %s, side %s, price: %s, quantity: %s\n",
		// 	orderUpdate.ClientOrderId,
		// 	binancesdk.SideType(orderUpdate.Side),
		// 	orderUpdate.Price,
		// 	orderUpdate.Volume,
		// )
	case "CANCELED":
		// logrus.Infof("[CANCELED]: ID: %s, side %s, price: %s, quantity: %s\n",
		// 	orderUpdate.ClientOrderId,
		// 	binancesdk.SideType(orderUpdate.Side),
		// 	orderUpdate.Price,
		// 	orderUpdate.Volume,
		// )
	case "FILLED":
		// logrus.Infof("[FILLED]: %+v\n", orderUpdate)
		a.binanceOrdersCh <- Order{
			ID:    orderUpdate.ClientOrderId,
			Price: stringToDecimal(orderUpdate.LatestPrice),
			Qty:   stringToDecimal(orderUpdate.FilledVolume),
		}

	case "PARTIALLY_FILLED":
		// logrus.Infof("[FILLED]: %+v\n", orderUpdate)
	default:
		// logrus.Infof("[FILLED]: %+v\n", orderUpdate)
	}
}

func (a *Account) profitLog(orderIds OrderIds) {
	openBinanceOrder, openMexcOrder, closeBinanceOrder, closeMexcOrder, ok := a.getOrders(orderIds)
	if !ok {
		logrus.Warn("订单错误")
		// binanceWsServiceRestartCh <- struct{}{}
		return
	}

	var (
		tusdProfit = decimal.Zero
		usdtProfit = decimal.Zero
	)

	switch orderIds.mode {
	case 1:
		tusdProfit = openBinanceOrder.Price.Sub(closeBinanceOrder.Price)
		usdtProfit = closeMexcOrder.Price.Sub(openMexcOrder.Price)
	case 2:
		tusdProfit = closeBinanceOrder.Price.Sub(openBinanceOrder.Price)
		usdtProfit = openMexcOrder.Price.Sub(closeMexcOrder.Price)
	default:
		panic("Invalid mode")
	}

	msg := fmt.Sprintf("模式%d \n[开仓]: BTC/TUSD: %s BTC/USDT: %s\n[平仓]: BTC/TUSD: %s BTC/USDT: %s\n[实际盈利] BTC/TUSD: %s BTC/USDT: %s\n[合计实际结果] %s",
		orderIds.mode,
		openBinanceOrder.Price.String(), openMexcOrder.Price.String(),
		closeBinanceOrder.Price.String(), closeMexcOrder.Price.String(),
		tusdProfit.String(), usdtProfit.String(),
		tusdProfit.Add(usdtProfit).String())

	logrus.Infof(msg)
}

func (a *Account) getOrders(orderIds OrderIds) (openBinanceOrder, openMexcOrder, closeBinanceOrder, closeMexcOrder Order, ok bool) {
	var (
		wg          sync.WaitGroup
		ticker      = time.NewTicker(time.Millisecond * 10)
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10000)
		cnt         atomic.Int64
	)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if openBinanceOrder, ok = a.getBinanceOrderWithContext(orderIds.OpenBinanceID, ticker, ctx); ok {
			cnt.Add(1)
		} else {
			logrus.Warn("获取binance开仓订单失败")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if openMexcOrder, ok = a.getMexcOrderWithContext(orderIds.OpenMexcID, ticker, ctx); ok {
			cnt.Add(1)
		} else {
			logrus.Warn("获取mexc开仓订单失败")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if closeBinanceOrder, ok = a.getBinanceOrderWithContext(orderIds.CloseBinanceID, ticker, ctx); ok {
			cnt.Add(1)
		} else {
			logrus.Warn("获取binance平仓订单失败")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if closeMexcOrder, ok = a.getMexcOrderWithContext(orderIds.CloseMexcID, ticker, ctx); ok {
			cnt.Add(1)
		} else {
			logrus.Warn("获取mexc平仓订单失败")
		}
	}()

	wg.Wait()
	a.clearOrders()

	if cnt.Load() != 4 {
		return openBinanceOrder, openMexcOrder, closeBinanceOrder, closeMexcOrder, false
	} else {
		return openBinanceOrder, openMexcOrder, closeBinanceOrder, closeMexcOrder, true
	}

}

func (a *Account) clearOrders() {
	a.L.Lock()
	defer a.L.Unlock()
	a.binanceOrders = make(map[string]Order)
	a.mexcOrders = make(map[string]Order)
}

func (a *Account) getBinanceOrderWithContext(id string, ticker *time.Ticker, ctx context.Context) (Order, bool) {
	for {
		select {
		case <-ctx.Done():
			return Order{}, false
		case <-ticker.C:
			if order, ok := a.getBinanceOrder(id); ok {
				return order, ok
			}
		}
	}
}
func (a *Account) getMexcOrderWithContext(id string, ticker *time.Ticker, ctx context.Context) (Order, bool) {
	for {
		select {
		case <-ctx.Done():
			return Order{}, false
		case <-ticker.C:
			if order, ok := a.getMexcOrder(id); ok {
				return order, ok
			}
		}
	}
}

func (a *Account) getBinanceOrder(id string) (Order, bool) {
	a.L.RLock()
	defer a.L.RUnlock()

	v, ok := a.binanceOrders[id]
	return v, ok
}

func (a *Account) getMexcOrder(id string) (Order, bool) {
	a.L.RLock()
	defer a.L.RUnlock()

	v, ok := a.mexcOrders[id]
	return v, ok
}

func stringToDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
