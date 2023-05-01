package mexc

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	baseWsApiURL = "wss://wbs.mexc.com/ws"
)

func getWsEndpoint() string {
	return baseWsApiURL
}

type WsApiRequest struct {
	Method WsApiMethod `json:"method"`
	Params []string    `json:"params"`
}

func WsDepthServe(symbol string, wsHandler WsHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
	req, err := json.Marshal(WsApiRequest{
		Method: SUBSCRIPTION,
		Params: []string{
			fmt.Sprintf("spot@public.increase.depth.v3.api@%s", strings.ToUpper(symbol)),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Set up the connection
	cfg := newWsConfig(req, getWsEndpoint())

	return wsServe(cfg, wsHandler, errHandler)
}

func WsPartialDepthServe(symbol, level string, wsHandler WsHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
	req, err := json.Marshal(WsApiRequest{
		Method: SUBSCRIPTION,
		Params: []string{
			fmt.Sprintf("spot@public.limit.depth.v3.api@%s@%s", strings.ToUpper(symbol), level),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Set up the connection
	cfg := newWsConfig(req, getWsEndpoint())

	return wsServe(cfg, wsHandler, errHandler)
}

type WsBookTickerEvent struct {
	Params    string `json:"params"`
	Data      `json:"d"`
	Symbol    string `json:"s"`
	Timestamp int64  `json:"t"`
}

type Data struct {
	AskQty   string `json:"A"` //卖单最优挂单数量
	BidQty   string `json:"B"` //买单最优挂单数量
	AskPrice string `json:"a"` //卖单最优挂单价格`
	BidPrice string `json:"b"` //买单最优挂单价格
}

type WsBookTickerHandler func(event *WsBookTickerEvent)

func WsBookTickerServe(symbol string, handler WsBookTickerHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
	req, err := json.Marshal(WsApiRequest{
		Method: SUBSCRIPTION,
		Params: []string{
			fmt.Sprintf("spot@public.bookTicker.v3.api@%s", strings.ToUpper(symbol)),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Set up the connection
	cfg := newWsConfig(req, getWsEndpoint())

	wsHandler := func(msg []byte) {
		event := WsBookTickerEvent{}
		err := json.Unmarshal(msg, &event)
		if err != nil {
			panic(err)
		}
		handler(&event)
	}

	return wsServe(cfg, wsHandler, errHandler)
}
