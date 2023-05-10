package binance

import (
	"context"
	"fmt"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/config"
)

func newContext() context.Context {
	return context.Background()
}

//	WS ListenKey
//
// 1 生成 Listen Key  Create a ListenKey
func CreateListenKey() string {

	client := binancesdk.NewClient(config.Config.BinanceApiKey, config.Config.BinanceSecretKey)

	response, err := client.NewStartUserStreamService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}
	listenKey := response
	return listenKey
}

// 2 延长 Listen Key 有效期  Keep-alive a ListenKey
func KeepListenKey(listenKey string) interface{} {
	client := binancesdk.NewClient(config.Config.BinanceApiKey, config.Config.BinanceSecretKey)
	err := client.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(newContext())
	if err != nil {
		fmt.Println("Current listenKey: ", listenKey)
		fmt.Println(err)
		return nil
	}
	return nil
}

// 3 关闭 Listen Key  Close a ListenKey
func CloseListenKey(listenKey string) interface{} {
	client := binancesdk.NewClient(config.Config.BinanceApiKey, config.Config.BinanceSecretKey)
	err := client.NewCloseUserStreamService().ListenKey(listenKey).Do(newContext())
	if err != nil {
		fmt.Println("Current listenKey: ", listenKey)
		fmt.Println(err)
		return nil
	}
	return nil
}
