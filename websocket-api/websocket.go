package websocket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WsHandler handle raw websocket message
type WsHandler func(message []byte)

// ErrHandler handles errors
type ErrHandler func(err error)

// WsConfig webservice configuration
type WsConfig struct {
	Endpoint string
}

func newWsConfig(endpoint string) *WsConfig {
	return &WsConfig{
		Endpoint: endpoint,
	}
}

var wsServe = func(cfg *WsConfig, handler WsHandler, errHandler ErrHandler) (msgC chan []byte, doneC, stopC chan struct{}, err error) {
	Dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  45 * time.Second,
		EnableCompression: false,
	}

	c, _, err := Dialer.Dial(cfg.Endpoint, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	//BUG:c.SetReadLimit(655350)
	msgC = make(chan []byte)
	doneC = make(chan struct{})
	stopC = make(chan struct{})
	go func() {
		// This function will exit either on error from
		// websocket.Conn.ReadMessage or when the stopC channel is
		// closed by the client.
		defer func() {
			close(doneC)
		}()
		if WebsocketKeepalive {
			keepAlive(c, WebsocketTimeout)
		}
		// Wait for the stopC channel to be closed.  We do that in a
		// separate goroutine because ReadMessage is a blocking
		// operation.
		silent := false
		go func() {
			select {
			case <-stopC:
				silent = true
			case <-doneC:
			}
			c.Close()
		}()

		// Send messages to binance.
		go func() {
			for {
				message := <-msgC

				c.WriteMessage(websocket.TextMessage, message)
			}
		}()

		// Receive messages from binance.
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if !silent {
					errHandler(err)
				}
				return
			}
			handler(message)
		}
	}()
	return
}

func keepAlive(c *websocket.Conn, timeout time.Duration) {
	ticker := time.NewTicker(timeout)

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		defer ticker.Stop()
		for {
			deadline := time.Now().Add(10 * time.Second)
			err := c.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				return
			}
			<-ticker.C
			if time.Since(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}

func signature(src, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(src))
	return hex.EncodeToString(mac.Sum(nil))
}

func currentTimestamp() int64 {
	return int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
}
