package mexc

const (
	baseHttpURL  = "https://api.mexc.com"
	ListenKeyApi = "/api/v3/userDataStream"
)

func getHttpEndpoint() string {
	return baseHttpURL
}
func getNewListenKeyApi() string {
	return ListenKeyApi
}

// WS ListenKey
//
// 生成 Listen Key  Create a ListenKey
func CreateListenKey(jsonParams string) interface{} {
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
