package websocket

import (
	"encoding/json"
	"log"
	"time"
)

const (
	baseWsApiURL        = "wss://ws-api.binance.com:443/ws-api/v3"
	baseWsApiTestnetURL = "wss://testnet.binance.vision/ws-api/v3"
)

var (
	// WebsocketTimeout is an interval for sending ping/pong messages if WebsocketKeepalive is enabled
	WebsocketTimeout = time.Second * 60
	// WebsocketKeepalive enables sending ping/pong messages to check the connection stability
	WebsocketKeepalive = false

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
	events map[string]WsApiMethod
}

func NewWebsocketServiceManager() *WebsocketServiceManager {
	return &WebsocketServiceManager{
		events: make(map[string]WsApiMethod),
	}
}

func (w *WebsocketServiceManager) StartWsApi(wsHandler WsHandler, errHandler ErrHandler) (chan *WsApiRequest, chan *WsApiEvent, chan struct{}, chan struct{}) {
	var (
		requestCh  = make(chan *WsApiRequest)
		responseCh = make(chan *WsApiEvent)
		msgC       chan []byte
		doneC      chan struct{}
		stopC      chan struct{}
		err        error
		cnt        = 1
	)

	// Set up the connection
	endpoint := getWsEndpoint()
	cfg := newWsConfig(endpoint)

	for {
		msgC, doneC, stopC, err = wsServe(cfg, wsHandler, errHandler)
		if err == nil {
			break
		}

		log.Printf("[ERROR] Failed to connect to websocket server: %v, reconnect server: %d", err, cnt)
		cnt++
		continue
	}
	log.Println("Connect to websocket server successfully.")

	go func() {
		for {
			wsApiRequest := <-requestCh
			msg, err := json.Marshal(wsApiRequest)
			if err != nil {
				log.Println("[ERROR] Failed to marshal wsApiRequest:", err)
			}
			// log.Println(string(msg))

			msgC <- msg
			w.events[wsApiRequest.ID] = wsApiRequest.Method
			log.Println("Send wsApiRequest successfully: ", wsApiRequest)
		}
	}()

	return requestCh, responseCh, doneC, stopC
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
