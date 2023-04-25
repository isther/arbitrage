package binancemexc

import (
	"testing"

	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexcsdk"
	"github.com/isther/arbitrage/websocket-api/mexc"
)

func TestMexcOrder(t *testing.T) {
	config.Load("../config.yaml")
	var (
		spot      = mexcsdk.NewSpot(&config.Config.MexcApiKey, &config.Config.MexcSecretKey)
		options   = make(map[string]string)
		symbol    = "BTCUSDT"
		side      = string(mexc.SideTypeBuy)
		orderType = string(mexc.OrderTypeMarket)
	)
	// options:{
	// 		timeInForce,
	// 		quantity,
	// 		quoteOrderQty,
	// 		price,
	// 		newClientOrderId,
	// 		stopPrice,
	// 		icebergQty,
	// 		newOrderRespType,
	// 		recvWindow
	// }
	options["timeInForce"] = string(mexc.TimeInForceTypeGTC)
	options["quantity"] = "0.001"

	res := spot.NewOrderTest(&symbol, &side, &orderType, options)
	t.Log(res)
}
