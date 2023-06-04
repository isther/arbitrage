package mexc

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

type WsPrivateAccountEvent struct {
	Params    string `json:"params"`
	Data      `json:"d"`
	Timestamp int64 `json:"t"`
}

type WsPrivateDealsEvent struct {
	Params    string `json:"params"`
	DealsData `json:"d"`
	Symbol    string `json:"s"`
	Timestamp int64  `json:"t"`
}

type DealsData struct {
	RemainAmount   string `json:"A"`  // 实际剩余金额
	RemainQty      string `json:"V"`  // 实际剩余数量
	SideType       int    `json:"S"`  // 订单方向
	CntOrderAmount string `json:"a"`  // 下单总金额
	OrderId        string `json:"i"`  // 订单ID
	Status         int    `json:"s"`  // 订单状态
	Price          string `json:"p"`  // 下单价格
	Qty            string `json:"v"`  // 下单数量
	CntQty         string `json:"cv"` // 累计成交数量
	CntAmount      string `json:"ca"` // 累计成交金额
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
//
//	func WsAccountInfoServe(handler WsPrivateAccountHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
//		listenkey := CreateListenKey()
//		if strings.TrimSpace(listenkey) == "" {
//			return nil, nil, errors.New("listenkey is empty")
//		}
//
//		go func() {
//			// 每30秒发送一个PUT
//			time.Sleep(30 * time.Second)
//			params := fmt.Sprintf(`{"listenKey": "%s"}`, listenkey)
//			KeepListenKey(params)
//		}()
//
//		// 账户信息推送
//		req, err := json.Marshal(WsApiRequest{
//			Method: SUBSCRIPTION,
//			Params: []string{
//				fmt.Sprintf("spot@private.account.v3.api"),
//			},
//		})
//		if err != nil {
//			return nil, nil, err
//		}
//
//		// Set up the connection
//		cfg := newWsConfig(req, getWsEndpoint()+"?listenKey="+listenkey)
//
//		wsHandler := func(msg []byte) {
//			event := WsPrivateAccountEvent{}
//			err := json.Unmarshal(msg, &event)
//			if err != nil {
//				panic(err)
//			}
//			handler(&event)
//		}
//
//		return wsServe(cfg, wsHandler, errHandler)
//
// }

func StartWsDealsInfoServer(listenkey string, handler WsPrivateDealsHandler, errHandler ErrHandler) (
	chan struct{},
	chan struct{},
) {
	var (
		err   error
		doneC chan struct{}
		stopC chan struct{}
	)
	for {
		doneC, stopC, err = WsDealsInfoServe(listenkey, handler, errHandler)
		if err == nil {
			break
		}
		logrus.Error(err)
		time.Sleep(time.Duration(reconnectMexcAccountInfoSleepDuration) * time.Millisecond)
	}
	logrus.Debug("Connect to mexc deals info websocket server successfully.")

	return doneC, stopC
}

func WsDealsInfoServe(listenkey string, handler WsPrivateDealsHandler, errHandler ErrHandler) (chan struct{}, chan struct{}, error) {
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
