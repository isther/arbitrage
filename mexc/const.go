package mexc

const (
	baseHttpURL   = "https://api.mexc.com"
	listenKeyApi  = "/api/v3/userDataStream"
	serverTimeApi = "/api/v3/time"
)

type WsApiMethod string

// SideType define side type of order
type SideType string

// OrderType define order type
type OrderType string

// TimeInForceType define time in force type of order
type TimeInForceType string

const (
	SUBSCRIPTION WsApiMethod = "SUBSCRIPTION"

	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"

	OrderTypeLimit           OrderType = "LIMIT"
	OrderTypeMarket          OrderType = "MARKET"
	OrderTypeStopLoss        OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"

	TimeInForceTypeGTC TimeInForceType = "GTC"
	TimeInForceTypeIOC TimeInForceType = "IOC"
	TimeInForceTypeFOK TimeInForceType = "FOK"
)

func getHttpEndpoint() string {
	return baseHttpURL
}
func getNewListenKeyApi() string {
	return listenKeyApi
}

func getServerTimeApi() string {
	return serverTimeApi
}
