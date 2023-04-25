package binance

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type ORDER_PARME string

type WsApiOrderTradeParams map[ORDER_PARME]interface{}

const (
	PARAM_SYMBOL              ORDER_PARME = "symbol"
	PARAM_SIDE                ORDER_PARME = "side"
	PARAM_TYPE                ORDER_PARME = "type"
	PARAM_TIME_IN_FORCE       ORDER_PARME = "timeInForce"
	PARAM_PRICE               ORDER_PARME = "price"
	PARAM_QTY                 ORDER_PARME = "quantity"
	PARAM_QUOTE_ORDER_QTY     ORDER_PARME = "quoteOrderQty"
	PARAM_NEW_CLIENT_ORDER_ID ORDER_PARME = "newClientOrderId"
	PARAM_NEW_ORDER_RESP_TYPE ORDER_PARME = "newOrderRespType"
	PARAM_API_KEY             ORDER_PARME = "apiKey"
	PARAM_SIGNATURE           ORDER_PARME = "signature"
	PARAM_TIMESTAMP           ORDER_PARME = "timestamp"
	PARAM_RECV_WINDOW         ORDER_PARME = "recvWindow"
)

type WsApiOrderTrade struct {
	Symbol           string           `json:"symbol"`
	Side             SideType         `json:"side"`
	OrderType        OrderType        `json:"type"`
	TimeInForce      TimeInForceType  `json:"timeInForce"`
	Price            string           `json:"price"`
	Quantity         string           `json:"quantity"`
	QuoteOrderQty    string           `json:"quoteOrderQty"`
	NewClientOrderID string           `json:"newClientOrderId"`
	NewOrderRespType NewOrderRespType `json:"newOrderRespType"`
	ApiKey           string           `json:"apiKey"`
	Signature        string           `json:"signature"`
	Timestamp        string           `json:"timestamp"`
}

// WsApiOrderTradeEvent define create order response
type WsApiOrderTradeEvent struct {
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	OrderListID   int64  `json:"orderListId"`
	ClientOrderID string `json:"clientOrderId"`
	TransactTime  int64  `json:"transactTime"`

	Price                    string `json:"price"`
	OrigQuantity             string `json:"origQty"`
	ExecutedQuantity         string `json:"executedQty"`
	CummulativeQuoteQuantity string `json:"cummulativeQuoteQty"`

	Status                  OrderStatusType `json:"status"`
	TimeInForce             TimeInForceType `json:"timeInForce"`
	Type                    OrderType       `json:"type"`
	Side                    SideType        `json:"side"`
	WorkingTime             int64           `json:"workingTime"`
	SelfTradePreventionMode string          `json:"selfTradePreventionMode"`

	// for order response is set to FULL
	Fills []*Fill `json:"fills"`
}

// Fill may be returned in an array of fills in a CreateOrderResponse.
type Fill struct {
	TradeID         int    `json:"tradeId"`
	Price           string `json:"price"`
	Quantity        string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}

func NewOrderTrade(wsApiOrderTradeParams WsApiOrderTradeParams) *WsApiRequest {
	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: OrderTrade,
		Params: wsApiOrderTradeParams,
	}
}

func NewOrderTradeTest(wsApiOrderTradeParams WsApiOrderTradeParams) *WsApiRequest {
	return &WsApiRequest{
		ID:     uuid.New().String(),
		Method: OrderTradeTest,
		Params: wsApiOrderTradeParams,
	}
}

func NewOrderTradeParmes(apiKey string) WsApiOrderTradeParams {
	var w = make(WsApiOrderTradeParams)
	w[PARAM_API_KEY] = apiKey
	return w
}

func (w WsApiOrderTradeParams) Timestamp(timestamp int64) WsApiOrderTradeParams {
	w[PARAM_TIMESTAMP] = timestamp
	return w
}

func (w WsApiOrderTradeParams) RecvWindow(recvWindow string) WsApiOrderTradeParams {
	w[PARAM_RECV_WINDOW] = recvWindow
	return w
}

func (w WsApiOrderTradeParams) TimeInForce(timeInForce TimeInForceType) WsApiOrderTradeParams {
	w[PARAM_TIME_IN_FORCE] = timeInForce
	return w
}

func (w WsApiOrderTradeParams) ApiKey(apiKey string) WsApiOrderTradeParams {
	w[PARAM_API_KEY] = apiKey
	return w
}

func (w WsApiOrderTradeParams) Symbol(symbol string) WsApiOrderTradeParams {
	w[PARAM_SYMBOL] = symbol
	return w
}

func (w WsApiOrderTradeParams) Side(side SideType) WsApiOrderTradeParams {
	w[PARAM_SIDE] = side
	return w
}

func (w WsApiOrderTradeParams) OrderType(orderType OrderType) WsApiOrderTradeParams {
	w[PARAM_TYPE] = orderType
	return w
}

func (w WsApiOrderTradeParams) Price(price string) WsApiOrderTradeParams {
	w[PARAM_PRICE] = price
	return w
}

func (w WsApiOrderTradeParams) Quantity(quantity string) WsApiOrderTradeParams {
	w[PARAM_QTY] = quantity
	return w
}

func (w WsApiOrderTradeParams) QuoteOrderQty(quoteOrderQty string) WsApiOrderTradeParams {
	w[PARAM_QUOTE_ORDER_QTY] = quoteOrderQty
	return w
}

func (w WsApiOrderTradeParams) NewClientOrderID(newClientOrderID string) WsApiOrderTradeParams {
	w[PARAM_NEW_CLIENT_ORDER_ID] = newClientOrderID
	return w
}

func (w WsApiOrderTradeParams) NewOrderRespType(newOrderRespType NewOrderRespType) WsApiOrderTradeParams {
	w[PARAM_NEW_ORDER_RESP_TYPE] = newOrderRespType
	return w
}

func (w WsApiOrderTradeParams) Signature(secretKey string) WsApiOrderTradeParams {
	w.Timestamp(currentTimestamp() - TimeOffset)

	params := url.Values{}
	for k, v := range w {
		params.Set(string(k), fmt.Sprintf("%v", v))
	}

	w[PARAM_SIGNATURE] = signature(params.Encode(), secretKey)
	return w
}
