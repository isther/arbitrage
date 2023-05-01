package mexc

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

func MexcBTCBuy(cookie, price, quantity string) (string, error) {
	return sendMexcOrder(cookie, "BTC", price, quantity, string(SideTypeBuy))
}

func MexcBTCSell(cookie, price, quantity string) (string, error) {
	return sendMexcOrder(cookie, "BTC", price, quantity, string(SideTypeSell))
}

func sendMexcOrder(cookie, currency, price, quantity, OrderType string) (string, error) {
	client := &http.Client{}
	song := make(map[string]string)
	song["currency"] = currency
	song["market"] = "USDT"
	song["tradeType"] = OrderType
	song["quantity"] = quantity
	song["price"] = price
	song["orderType"] = "LIMIT_ORDER"
	bytesData, _ := json.Marshal(song)

	req, err := http.NewRequest("POST", "https://www.mexc.com/api/platform/spot/order/place",
		bytes.NewBuffer([]byte(bytesData)))
	if err != nil {
		log.Println(err)
		return "", err
	}
	var stimep = strconv.Itoa(int(time.Now().Unix() * 1000))
	var sign = md5Encrypt(stimep)

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("content-length", "119")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("cookie", cookie)
	req.Header.Set("language", "zh-CN")
	req.Header.Set("origin", "https://www.mexc.com")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("referer", "https://www.mexc.com/zh-CN/exchange/BTC_USDT?_from=header")
	req.Header.Set("sec-ch-ua", "\"Chromium\";v=\"112\", \"Google Chrome\";v=\"112\", \"Not:A-Brand\";v=\"99\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")
	req.Header.Set("x-mxc-nonce", stimep)
	req.Header.Set("x-mxc-sign", sign)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}

	// str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	str := string(content)
	return str, nil

}

func md5Encrypt(txt string) string {
	m5 := md5.New()
	m5.Write([]byte(txt))
	txtHash := hex.EncodeToString(m5.Sum(nil))
	return txtHash
}
