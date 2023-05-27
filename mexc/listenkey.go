package mexc

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
)

// WS ListenKey
//
// 生成 Listen Key  Create a ListenKey

func CreateListenKey() string {
	var params string = ""
	resp := createListenKey(params)
	newListenKey, ok := resp.(*resty.Response)
	if !ok {
		panic("create listenkey failed")
	}
	var listenKeyResponse ListenKeyResponse
	if err := json.Unmarshal(newListenKey.Body(), &listenKeyResponse); err != nil {
		panic(fmt.Sprintf("marshal listenkey failed: %+v", newListenKey))
	}
	return listenKeyResponse.ListenKey
}

func createListenKey(jsonParams string) interface{} {
	requestUrl := getHttpEndpoint() + getNewListenKeyApi()
	// fmt.Println("requestUrl:", requestUrl)
	response := PrivatePost(requestUrl, jsonParams)
	return response
}

// 延长 Listen Key 有效期  Keep-alive a ListenKey
func KeepListenKey(jsonParams string) interface{} {
	requestUrl := getHttpEndpoint() + getNewListenKeyApi()
	// fmt.Println("requestUrl:", requestUrl)
	response := PrivatePut(requestUrl, jsonParams)
	return response
}

// 关闭 Listen Key  Close a ListenKey
func CloseListenKey(jsonParams string) interface{} {
	requestUrl := getHttpEndpoint() + getNewListenKeyApi()
	// fmt.Println("requestUrl:", requestUrl)
	response := PrivateDelete(requestUrl, jsonParams)
	return response
}
