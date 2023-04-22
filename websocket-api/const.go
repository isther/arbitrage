package websocket

type WsApiMethod string

// SideType define side type of order
type SideType string

// OrderType define order type
type OrderType string

// TimeInForceType define time in force type of order
type TimeInForceType string

// NewOrderRespType define response JSON verbosity
type NewOrderRespType string

// OrderStatusType define order status type
type OrderStatusType string

// RateLimitType define the rate limitation types
// see https://github.com/binance/binance-spot-api-docs/blob/master/rest-api.md#enum-definitions
type RateLimitType string

// RateLimitInterval define the rate limitation intervals
type RateLimitInterval string

// Global enums
const (
	Ping           WsApiMethod = "ping"
	ServerTime     WsApiMethod = "time"
	ExchangeInfo   WsApiMethod = "exchangeInfo"
	AccountStatus  WsApiMethod = "account.status"
	OrderTrade     WsApiMethod = "order.place"
	OrderTradeTest WsApiMethod = "order.test"

	SideTypeBuy  SideType = "BUY"
	SideTypeSell SideType = "SELL"

	OrderTypeLimit           OrderType = "LIMIT"
	OrderTypeMarket          OrderType = "MARKET"
	OrderTypeLimitMaker      OrderType = "LIMIT_MAKER"
	OrderTypeStopLoss        OrderType = "STOP_LOSS"
	OrderTypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	OrderTypeTakeProfit      OrderType = "TAKE_PROFIT"
	OrderTypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"

	TimeInForceTypeGTC TimeInForceType = "GTC"
	TimeInForceTypeIOC TimeInForceType = "IOC"
	TimeInForceTypeFOK TimeInForceType = "FOK"

	NewOrderRespTypeACK    NewOrderRespType = "ACK"
	NewOrderRespTypeRESULT NewOrderRespType = "RESULT"
	NewOrderRespTypeFULL   NewOrderRespType = "FULL"

	OrderStatusTypeNew             OrderStatusType = "NEW"
	OrderStatusTypePartiallyFilled OrderStatusType = "PARTIALLY_FILLED"
	OrderStatusTypeFilled          OrderStatusType = "FILLED"
	OrderStatusTypeCanceled        OrderStatusType = "CANCELED"
	OrderStatusTypePendingCancel   OrderStatusType = "PENDING_CANCEL"
	OrderStatusTypeRejected        OrderStatusType = "REJECTED"
	OrderStatusTypeExpired         OrderStatusType = "EXPIRED"

	RateLimitTypeRequestWeight RateLimitType = "REQUEST_WEIGHT"
	RateLimitTypeOrders        RateLimitType = "ORDERS"
	RateLimitTypeRawRequests   RateLimitType = "RAW_REQUESTS"

	RateLimitIntervalSecond RateLimitInterval = "SECOND"
	RateLimitIntervalMinute RateLimitInterval = "MINUTE"
	RateLimitIntervalDay    RateLimitInterval = "DAY"
)
