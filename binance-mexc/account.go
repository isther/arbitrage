package binancemexc

import (
	"fmt"
	"os"
	"strings"
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

type Account struct {
	StableSymbolAsset  Asset
	BinanceSymbolAsset Asset
	MexcSymbolAsset    Asset

	OrderIDsCh    chan OrderIds
	BinanceOrders map[string]Order
	MexcOrders    map[string]Order
	L             sync.RWMutex
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
	Price decimal.Decimal
	Qty   decimal.Decimal
}

func NewAccount(symbolPair SymbolPair) *Account {
	return &Account{
		StableSymbolAsset:  Asset{Symbol: symbolPair.StableCoinSymbol},
		BinanceSymbolAsset: Asset{Symbol: symbolPair.BinanceSymbol},
		MexcSymbolAsset:    Asset{Symbol: symbolPair.MexcSymbol},
		OrderIDsCh:         make(chan OrderIds),
		BinanceOrders:      make(map[string]Order),
		MexcOrders:         make(map[string]Order),
	}
}

func (a *Account) Start() {
	go func() {
		for {
			orderIDs := <-a.OrderIDsCh
			a.profitLog(orderIDs)

			number++
			if config.Config.Number == number {
				logrus.Warn("软件已停止")
				time.Sleep(3001 * time.Millisecond)
				os.Exit(-1)
			}
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
			restartCh = make(chan struct{})
			listenKey = binance.CreateListenKey()
		)
		defer binance.CloseListenKey(listenKey)

		go func() {
			for {
				time.Sleep(25 * time.Minute)
				binance.KeepListenKey(listenKey)
			}
		}()

		for {
			_, _ = binance.StartWsUserData(
				listenKey,
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
		var restartCh = make(chan struct{})
		for {
			_, _ = mexc.StartWsDealsInfoServer(
				func(event *mexc.WsPrivateDealsEvent) {
					// logrus.WithFields(logrus.Fields{"server": "mexc account"}).Infof("%+v", event)
					a.L.Lock()
					defer a.L.Unlock()

					if strings.TrimSpace(event.Price) == "" {
						return
					}

					a.MexcOrders[event.DealsData.OrderId] = Order{
						Price: stringToDecimal(event.Price),
						Qty:   stringToDecimal(event.Qty),
					}
				},
				func(err error) {
					if err != nil {
						logrus.WithFields(logrus.Fields{"server": "mexc account"}).Error(err)
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
		if accountUpdate.Asset == a.StableSymbolAsset.Symbol {
		}
		if accountUpdate.Asset == a.BinanceSymbolAsset.Symbol {
		}
	}

	logrus.Infof("账户余额更新: %v: %v %v\n %v: %v %v\n %v: %v %v\n",
		a.StableSymbolAsset.Symbol, a.StableSymbolAsset.Free, a.StableSymbolAsset.Locked,
		a.BinanceSymbolAsset.Symbol, a.BinanceSymbolAsset.Free, a.BinanceSymbolAsset.Locked,
		a.MexcSymbolAsset.Symbol, a.MexcSymbolAsset.Free, a.MexcSymbolAsset.Locked)
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
		a.L.Lock()
		defer a.L.Unlock()
		a.BinanceOrders[orderUpdate.ClientOrderId] = Order{
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
		logrus.Error("订单错误")
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

func (a *Account) getOrders(orderIds OrderIds) (Order, Order, Order, Order, bool) {
	var (
		cnt = 0
		f   = func() {
			time.Sleep(100 * time.Millisecond)
			cnt++
		}
	)

	for cnt < 100 {
		openBinanceOrder, ok := a.getBinanceOrder(orderIds.OpenBinanceID)
		if !ok {
			f()
			continue
		}

		openMexcOrder, ok := a.getMexcOrder(orderIds.OpenMexcID)
		if !ok {
			f()
			continue
		}

		closeBinanceOrder, ok := a.getBinanceOrder(orderIds.CloseBinanceID)
		if !ok {
			f()
			continue
		}

		closeMexcOrder, ok := a.getMexcOrder(orderIds.CloseMexcID)
		if !ok {
			f()
			continue
		}

		return openBinanceOrder, openMexcOrder, closeBinanceOrder, closeMexcOrder, true

	}
	return Order{}, Order{}, Order{}, Order{}, false
}

func (a *Account) getBinanceOrder(id string) (Order, bool) {
	a.L.RLock()
	defer a.L.RUnlock()

	v, ok := a.BinanceOrders[id]
	return v, ok
}

func (a *Account) getMexcOrder(id string) (Order, bool) {
	a.L.RLock()
	defer a.L.RUnlock()

	v, ok := a.MexcOrders[id]
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
