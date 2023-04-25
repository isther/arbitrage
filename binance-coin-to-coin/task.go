package arbitrage

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/isther/arbitrage/websocket-api/binance"
	"github.com/shopspring/decimal"
)

type Task struct {
	apiKey    string
	secretKey string

	StableCoinA string // eg: TUSD
	TradeCoin   string // eg: BTC
	StablecoinB string // eg: USDT

	MinRatio decimal.Decimal
	MaxRatio decimal.Decimal

	qtyCh  chan struct{}
	stopCh chan struct{}
}

func NewArbitrageTask(
	apiKey, secretKey,
	stableCoinA, tradeCoin, stablecoinB string,
	minRatio, maxRatio float64) (*Task, error) {
	if strings.Compare(stableCoinA, stablecoinB) == 0 {
		return nil, errors.New("StablecoinA equal StablecoinB")
	}

	if strings.Compare(stableCoinA, stablecoinB) > 0 {
		stableCoinA, stablecoinB = stablecoinB, stableCoinA
	}

	return &Task{
		apiKey:      apiKey,
		secretKey:   secretKey,
		StableCoinA: stableCoinA,
		TradeCoin:   tradeCoin,
		StablecoinB: stablecoinB,
		qtyCh:       make(chan struct{}),
		stopCh:      make(chan struct{}),
		MinRatio:    decimal.NewFromFloat(minRatio),
		MaxRatio:    decimal.NewFromFloat(maxRatio),
	}, nil
}

func (t *Task) Run() {
	for {
		ticker := time.NewTicker(1 * time.Millisecond)
		select {
		case <-ticker.C:
			t.Trade()
		case <-t.stopCh:
			log.Println("Stop")
			return
		}
	}
}

func (t *Task) Trade() {
	tradeCoinStableCoinASymbolEvent, tradeCoinStableCoinBSymbolEvent, StableCoinAStableCoinBSymbolEvent := ArbitrageManagerInstance.GetBookTickerEvent(
		t.TradeCoin+t.StableCoinA,
		t.TradeCoin+t.StablecoinB,
		t.StableCoinA+t.StablecoinB,
	)

	// Price
	// AskPrice: Minimum sell pice
	// BidPrice: Maximum buy price

	// A->T Buy AskPrice
	// T->B Sell BidPrice
	// B->A Sell AskPrice
	taAskPrice, _ := decimal.NewFromString(tradeCoinStableCoinASymbolEvent.BestAskPrice)
	tbBidPrice, _ := decimal.NewFromString(tradeCoinStableCoinBSymbolEvent.BestBidPrice)
	abAskPrice, _ := decimal.NewFromString(StableCoinAStableCoinBSymbolEvent.BestAskPrice)
	if t.judgeRatio(false, taAskPrice, tbBidPrice, abAskPrice) {
		// TODO:Quantity
		// TA AskQty
		// taAskQty, _ := decimal.NewFromString(tradeCoinStableCoinASymbolEvent.BestAskQty)
		// TB BidQty
		// tbBidQty, _ := decimal.NewFromString(tradeCoinStableCoinBSymbolEvent.BestBidQty)
		// AB AskQty
		// abAskQty, _ := decimal.NewFromString(StableCoinAStableCoinBSymbolEvent.BestAskQty)

		// taToTQty := taAskQty
		// tbToTQty := tbBidQty.Mul(tbBidPrice).Div(abAskPrice)
		// abToTQty := abAskQty.Div(abAskPrice)

		var wg = sync.WaitGroup{}

		wg.Add(2)
		// TradeCoin + StableCoinA
		go func() {
			defer wg.Done()
			t.getOrderTrade(t.TradeCoin+t.StableCoinA, binance.SideTypeBuy, "")
		}()
		// TradeCoin + StableCoinB
		go func() {
			defer wg.Done()
			t.getOrderTrade(t.TradeCoin+t.StablecoinB, binance.SideTypeSell, "")
		}()

		wg.Wait()

		//StableCoinA + StablecoinB
		t.getOrderTrade(t.StableCoinA+t.StablecoinB, binance.SideTypeBuy, "")
		return
	}

	// B->T Buy AskPrice
	// T->A Sell BidPrice
	// A->B Sell BidPrice
	tbAskPrice, _ := decimal.NewFromString(tradeCoinStableCoinBSymbolEvent.BestAskPrice)
	taBidPrice, _ := decimal.NewFromString(tradeCoinStableCoinASymbolEvent.BestBidPrice)
	abBidPrice, _ := decimal.NewFromString(StableCoinAStableCoinBSymbolEvent.BestBidPrice)
	if t.judgeRatio(true, taBidPrice, tbAskPrice, abBidPrice) {
		// TODO:Quantity
		// tbAskQty, _ := decimal.NewFromString(tradeCoinStableCoinBSymbolEvent.BestAskQty)
		// taBidQty, _ := decimal.NewFromString(tradeCoinStableCoinASymbolEvent.BestBidQty)
		// abBidQty, _ := decimal.NewFromString(StableCoinAStableCoinBSymbolEvent.BestBidQty)

		var wg = sync.WaitGroup{}

		wg.Add(2)
		// TradeCoin + StableCoinA
		go func() {
			defer wg.Done()
			t.getOrderTrade(t.TradeCoin+t.StableCoinA, binance.SideTypeSell, "")
		}()

		// TradeCoin + StableCoinB
		go func() {
			defer wg.Done()
			t.getOrderTrade(t.TradeCoin+t.StablecoinB, binance.SideTypeBuy, "")
		}()

		wg.Wait()

		//StableCoinA + StablecoinB
		t.getOrderTrade(t.StableCoinA+t.StablecoinB, binance.SideTypeSell, "")
	}
}

// forward and reverse
func (t *Task) judgeRatio(reverse bool, taPrice, tbPrice, abPrice decimal.Decimal) bool {
	var (
		ratio decimal.Decimal
	)

	if !reverse {
		ratio = decimal.NewFromFloat32(1).Div(abPrice).
			Sub(
				taPrice.Div(
					tbPrice,
				),
			)
	} else {
		ratio = abPrice.
			Sub(
				tbPrice.Div(
					taPrice,
				),
			)
	}

	log.Println("Ratio:", ratio)
	return 0 <= ratio.Cmp(t.MinRatio) && ratio.Cmp(t.MaxRatio) <= 0
}

func (t *Task) getOrderTrade(symbol string, side binance.SideType, quantity string) *binance.WsApiRequest {
	params := binance.NewOrderTradeParmes(t.apiKey).
		NewOrderRespType(binance.NewOrderRespTypeRESULT).
		TimeInForce(binance.TimeInForceTypeGTC).
		Symbol(symbol).Side(side).
		OrderType(binance.OrderTypeMarket).Quantity(quantity).
		Signature(t.secretKey)

	if TestTrade {
		return binance.NewOrderTradeTest(params)
	}
	return binance.NewOrderTrade(params)
}

func decimalMin(a, b, c decimal.Decimal) decimal.Decimal {
	if a.Cmp(b) <= 0 {
		if a.Cmp(c) <= 0 {
			return a
		} else {
			return c
		}
	} else {
		if b.Cmp(c) <= 0 {
			return b
		} else {
			return c
		}
	}
}
