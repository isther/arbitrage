## 启动
为了方便启动和运行，使用Docker和docker-compose。直接执行make docker-cmd-run即可。

## 包说明
binance: 实现ws下单等。
mexc: 实现了抹茶ws信息订阅。
binance-coin-to-coin: 第一版币安套利的代码，可以忽略。
binance-mexc: 币安和抹茶套利的代码。

## 进度
目前完成了币安抹茶套利的核心代码，下单数量部分测试时写死为11，后续需要根据需求更改。

## 注意
binance-mexc/task.go文件getOrderBinanceTrade函数中有说明，在服务器上运行务必注释掉TimeInForce这行。
