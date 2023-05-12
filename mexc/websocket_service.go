package mexc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
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

type WsPrivateAccountEvent struct {
	Params    string `json:"params"`
	Data      `json:"d"`
	Timestamp int64 `json:"t"`
}

type WsPrivateDealsEvent struct {
	Params    string `json:"params"`
	Data      `json:"d"`
	Symbol    string `json:"s"`
	Timestamp int64  `json:"t"`
}

type Data struct {
	AskQty        string `json:"A"` //卖单最优挂单数量
	BidQty        string `json:"B"` //买单最优挂单数量
	AskPrice      string `json:"a"` //卖单最优挂单价格`
	BidPrice      string `json:"b"` //买单最优挂单价格
	OrderId       string `json:"i"`
}

type WsBookTickerHandler func(event *WsBookTickerEvent)
type WsPrivateAccountHandler func(event *WsPrivateAccountEvent)
type WsPrivateDealsHandler func(event *WsPrivateDealsEvent)

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

type ListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}

// 现货账户信息
func WsAccountInfoServe(handler WsPrivateAccountHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
	var params string = ""
	resp := CreateListenKey(params)
	NewListenKey, ok := resp.(*resty.Response)
	if !ok {
		return nil, nil, nil
	}
	var listenKeyResponse ListenKeyResponse
	if err := json.Unmarshal(NewListenKey.Body(), &listenKeyResponse); err != nil {
		return nil, nil, nil
	}
	listenkey := listenKeyResponse.ListenKey
	fmt.Println("new listenkey:", listenkey)

	go func() {
		// 每30秒发送一个PUT
		time.Sleep(30 * time.Second)
		params := "listenKey=" + listenkey
		KeepListenKey(params)
	}()

	// 账户信息推送
	req, err := json.Marshal(WsApiRequest{
		Method: SUBSCRIPTION,
		Params: []string{
			fmt.Sprintf("spot@private.account.v3.api"),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Set up the connection
	cfg := newWsConfig(req, getWsEndpoint()+"?listenKey="+listenkey)

	wsHandler := func(msg []byte) {
		event := WsPrivateAccountEvent{}
		err := json.Unmarshal(msg, &event)
		if err != nil {
			panic(err)
		}
		handler(&event)
	}

	return wsServe(cfg, wsHandler, errHandler)

}

// 现货账户成交
func WsDealsInfoServe(handler WsPrivateDealsHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
	var params string = ""
	resp := CreateListenKey(params)
	NewListenKey, ok := resp.(*resty.Response)
	if !ok {
		return nil, nil, nil
	}
	var listenKeyResponse ListenKeyResponse
	if err := json.Unmarshal(NewListenKey.Body(), &listenKeyResponse); err != nil {
		return nil, nil, nil
	}
	listenkey := listenKeyResponse.ListenKey
	fmt.Println("new listenkey:", listenkey)

	go func() {
		// 每30秒发送一个PUT
		time.Sleep(30 * time.Second)
		params := "listenKey=" + listenkey
		KeepListenKey(params)
	}()

	// 成交信息推送
	req, err := json.Marshal(WsApiRequest{
		Method: SUBSCRIPTION,
		Params: []string{
			fmt.Sprintf("spot@private.deals.v3.api"),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	// Set up the connection
	cfg := newWsConfig(req, getWsEndpoint()+"?listenKey="+listenkey)

	wsHandler := func(msg []byte) {
		event := WsPrivateDealsEvent{}
		err := json.Unmarshal(msg, &event)
		if err != nil {
			panic(err)
		}
		handler(&event)
	}

	return wsServe(cfg, wsHandler, errHandler)
}
