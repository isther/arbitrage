package binance

import (
	binancesdk "github.com/adshao/go-binance/v2"
)

func init() {
	// Keep ws alive
	WebsocketKeepalive = true
	binancesdk.WebsocketKeepalive = true
}
