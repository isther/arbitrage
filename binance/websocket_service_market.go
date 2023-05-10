package binance

import (
	"fmt"
	"net/url"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/google/uuid"
)

type WsApiPingParams struct{}

type WsApiPingEvent struct{}

func NewPing() *WsApiRequest {
	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: Ping,
		Params: WsApiPingParams{},
	}
}

type WsApiServerTimeParams struct{}

type WsApiServerTimeEvent struct {
	ServerTime int64 `json:"serverTime"`
}

func NewServerTime() *WsApiRequest {
	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: ServerTime,
		Params: WsApiServerTimeParams{},
	}
}

type WsApiExchangeInfoParams struct {
	// Symbol      string   `json:"symbol"`
	// Symbols []string `json:"symbols"`
	Permissions []string `json:"permissions"`
}

// ExchangeInfo exchange info
type WsApiExchangeInfoEvent struct {
	Timezone        string            `json:"timezone"`
	ServerTime      int64             `json:"serverTime"`
	RateLimits      []GlobalRateLimit `json:"rateLimits"`
	ExchangeFilters []interface{}     `json:"exchangeFilters"`
	Symbols         []Symbol          `json:"symbols"`
}

// Symbol market symbol
type Symbol struct {
	Symbol                     string                   `json:"symbol"`
	Status                     string                   `json:"status"`
	BaseAsset                  string                   `json:"baseAsset"`
	BaseAssetPrecision         int                      `json:"baseAssetPrecision"`
	QuoteAsset                 string                   `json:"quoteAsset"`
	QuotePrecision             int                      `json:"quotePrecision"`
	QuoteAssetPrecision        int                      `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int32                    `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int32                    `json:"quoteCommissionPrecision"`
	OrderTypes                 []string                 `json:"orderTypes"`
	IcebergAllowed             bool                     `json:"icebergAllowed"`
	OcoAllowed                 bool                     `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed bool                     `json:"quoteOrderQtyMarketAllowed"`
	IsSpotTradingAllowed       bool                     `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool                     `json:"isMarginTradingAllowed"`
	Filters                    []map[string]interface{} `json:"filters"`
	Permissions                []string                 `json:"permissions"`
}

func NewSpotExchangeInfo() *WsApiRequest {
	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: ExchangeInfo,
		Params: WsApiExchangeInfoParams{Permissions: []string{"SPOT"}},
	}
}

func NewKlineInfo(Symbol string, interval string) {
	var (
		params    = url.Values{}
		timestamp = currentTimestamp() - TimeOffset
	)

	params.Set(string(PARAM_TIMESTAMP), fmt.Sprintf("%v", timestamp))

	wsKlineHandler := func(event *binancesdk.WsKlineEvent) {
		fmt.Println(event)
	}
	errHandler := func(err error) {
		fmt.Println(err)
	}
	doneC, _, err := binancesdk.WsKlineServe(Symbol, interval, wsKlineHandler, errHandler)
	if err != nil {
		fmt.Println(err)
		return
	}
	<-doneC
	return
}
