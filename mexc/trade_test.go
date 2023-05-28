package mexc

import (
	"testing"

	"github.com/isther/arbitrage/config"
)

func TestMexcBuy(t *testing.T) {
	config.Load("../config.yaml")
	// 用户cookie
	var (
		cookie   = config.Config.MexcCookie
		price    = "30000"  // 购买价格
		quantity = "0.000350" // 购买数量
	)

	res, err := MexcBTCBuy(cookie, price, quantity)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}

func TestMexcSell(t *testing.T) {
	config.Load("../config.yaml")
	// 用户cookie
	var (
		cookie   = config.Config.MexcCookie
		price    = "25000"  // 出售价格
		quantity = "0.000350" // 出售数量
	)
	res, err := MexcBTCSell(cookie, price, quantity)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res)
}
