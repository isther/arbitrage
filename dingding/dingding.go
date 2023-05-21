package dingding

var (
	LogBot   *DingDingBot
	ErrorBot *DingDingBot
)

func Init(
	logBotAccessToken, logBotSecrect,
	errorBotAccessToken, errorBotSecrect string,
	chLen int,
) {
	LogBot = NewDingDingBot(logBotAccessToken, logBotSecrect, chLen)
	ErrorBot = NewDingDingBot(errorBotAccessToken, errorBotSecrect, chLen)

	go LogBot.Start()
	go ErrorBot.Start()
}
