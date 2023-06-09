package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const (
	baseWsApiURL        = "wss://ws-api.binance.com:9443/ws-api/v3"
	baseWsApiTestnetURL = "wss://testnet.binance.vision/ws-api/v3"
)

var (
	UseTestnet = false

	TimeOffset int64 = 0
)

func getWsEndpoint() string {
	if UseTestnet {
		return baseWsApiTestnetURL
	}
	return baseWsApiURL
}

type RateLimit struct {
	RateLimitType RateLimitType     `json:"rateLimitType"`
	Interval      RateLimitInterval `json:"interval"`
	IntervalNum   int               `json:"intervalNum"`
	Limit         int               `json:"limit"`
	Count         int               `json:"count"`
}

type GlobalRateLimit struct {
	RateLimitType RateLimitType     `json:"rateLimitType"`
	Interval      RateLimitInterval `json:"interval"`
	IntervalNum   int64             `json:"intervalNum"`
	Limit         int64             `json:"limit"`
}

type WsApiRequest struct {
	ID     string      `json:"id"`
	Method WsApiMethod `json:"method"`
	Params interface{} `json:"params"`
}

type WsApiEvent struct {
	ID                 string      `json:"id"`
	Status             int         `json:"status"`
	Result             interface{} `json:"result"`
	Ping               WsApiPingEvent
	ServerTime         WsApiServerTimeEvent
	ExchangeInfo       WsApiExchangeInfoEvent
	Account            AccountEvent
	OrderTradeResponse WsApiOrderTradeEvent
	RateLimits         []RateLimit `json:"rateLimits"`
	Error              WsApiError  `json:"error"`
}

type WsApiError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (wsApiEvent *WsApiEvent) parseEvent(method WsApiMethod) error {
	event, err := json.Marshal(wsApiEvent.Result)
	if err != nil {
		return err
	}

	switch method {
	case Ping:
		return json.Unmarshal(event, &wsApiEvent.Ping)
	case ServerTime:
		return json.Unmarshal(event, &wsApiEvent.ServerTime)
	case ExchangeInfo:
		return json.Unmarshal(event, &wsApiEvent.ExchangeInfo)
	case AccountStatus:
		return json.Unmarshal(event, &wsApiEvent.Account)
	case OrderTrade:
		return json.Unmarshal(event, &wsApiEvent.OrderTradeResponse)
	}
	return nil
}

type WebsocketServiceManager struct {
	events    map[string]WsApiMethod
	requestCh chan *WsApiRequest
}

func NewWebsocketServiceManager() *WebsocketServiceManager {
	return &WebsocketServiceManager{
		// requestCh: make(chan *WsApiRequest),
		events: make(map[string]WsApiMethod),
	}
}

func (w *WebsocketServiceManager) StartWsApi(wsHandler WsHandler, errHandler ErrHandler) (chan struct{}, chan struct{}) {
	Logger.Debug("Start binance websocket api service.")
	var (
		doneC chan struct{}
		stopC chan struct{}
		err   error
		cnt   = 1
	)

	// Set up the connection
	endpoint := getWsEndpoint()
	cfg := newWsConfig(endpoint)

	for {
		w.requestCh, doneC, stopC, err = wsServe(cfg, wsHandler, errHandler)
		if err == nil {
			break
		}

		Logger.Errorf("Failed to connect to websocket server: %v, reconnect server: %d", err, cnt)
		cnt++
		time.Sleep(500 * time.Millisecond)
	}
	Logger.Debug("Connect to websocket server successfully.")

	return doneC, stopC
}

func (w *WebsocketServiceManager) Send(req *WsApiRequest) {
	w.requestCh <- req
}

func (w *WebsocketServiceManager) ParseWsApiEvent(result []byte) (*WsApiEvent, WsApiMethod, error) {
	var (
		wsApiEvent = new(WsApiEvent)
		method     WsApiMethod
	)

	if err := json.Unmarshal(result, wsApiEvent); err != nil {
		return wsApiEvent, method, err
	}

	method = w.events[wsApiEvent.ID]

	err := wsApiEvent.parseEvent(method)
	return wsApiEvent, method, err
}

func signature(src, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(src))
	return hex.EncodeToString(mac.Sum(nil))
}

func currentTimestamp() int64 {
	return int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
}
