package mexc

import (
	"testing"

	"github.com/isther/arbitrage/config"
)

func TestMexcBuy(t *testing.T) {
	config.Load("../config.yaml")

	// 用户cookie
	var cookie = config.Config.MexcCookie
	var price = "30000"      // 购买价格
	var quantity = "0.00021" // 购买数量
	MexcBTCBuy(cookie, price, quantity)
}

func TestMexcSell(t *testing.T) {
	config.Load("../config.yaml")
	// 用户cookie
	var cookie = config.Config.MexcCookie
	var price = "25000"       // 出售价格
	var quantity = "0.000319" // 出售数量
	MexcBTCSell(cookie, price, quantity)
}
