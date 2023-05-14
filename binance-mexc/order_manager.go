package binancemexc

import (
	"sync"

	"github.com/shopspring/decimal"
)

type OrderPair struct {
	binancePrice decimal.Decimal
	binanceQty   decimal.Decimal
	mexcPrice    decimal.Decimal
	mexcQty      decimal.Decimal
}

type OrderManager struct {
	OpenBinanceOrderIdCh chan string
	OpenMexcOrderIdCh    chan string

	CloseBinanceOrderIdCh chan string
	CloseMexcOrderIdCh    chan string

	binanceOrder map[string]*OrderPair
	mexcOrder    map[string]*OrderPair

	L sync.RWMutex
}

func NewOrderManager() *OrderManager {
	return &OrderManager{
		OpenBinanceOrderIdCh: make(chan string),
		OpenMexcOrderIdCh:    make(chan string),

		CloseBinanceOrderIdCh: make(chan string),
		CloseMexcOrderIdCh:    make(chan string),
	}
}

func (o *OrderManager) Run() {
	go func() {
		for {
			openBinanceOrderId := <-o.OpenMexcOrderIdCh
			openMexcOrderId := <-o.OpenMexcOrderIdCh

			//TODO: log.pln
			o.CalOpen(
				o.getBinanceOrderPair(openBinanceOrderId),
				o.getMexcOrderPair(openMexcOrderId),
			)
		}
	}()

	go func() {
		for {
			closeBinanceOrderId := <-o.CloseBinanceOrderIdCh
			closeMexcOrderId := <-o.CloseMexcOrderIdCh

			o.CalClose(
				o.getBinanceOrderPair(closeBinanceOrderId),
				o.getMexcOrderPair(closeMexcOrderId),
			)
		}
	}()
}

func (o *OrderManager) CalOpen(openBinancePair, openMexcPair *OrderPair)    {}
func (o *OrderManager) CalClose(closeBinancePair, closeMexcPair *OrderPair) {}

func (o *OrderManager) getBinanceOrderPair(binanceOrderId string) *OrderPair {
	o.L.Lock()
	defer o.L.Unlock()

	if v, ok := o.binanceOrder[binanceOrderId]; ok {
		defer delete(o.binanceOrder, binanceOrderId)
		return v
	}

	return nil
}

func (o *OrderManager) getMexcOrderPair(mexcOrderId string) *OrderPair {
	o.L.Lock()
	defer o.L.Unlock()

	if v, ok := o.mexcOrder[mexcOrderId]; ok {
		defer delete(o.mexcOrder, mexcOrderId)
		return v
	}

	return nil
}
