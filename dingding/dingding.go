package dingding

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type DingDingBotHook struct {
	logBot   *DingDingBot
	errorBot *DingDingBot
}

func NewDingDingBotHook(
	logBotAccessToken, logBotSecrect,
	errorBotAccessToken, errorBotSecrect string,
	chLen int,
) *DingDingBotHook {
	hook := &DingDingBotHook{
		logBot:   NewDingDingBot(logBotAccessToken, logBotSecrect, chLen),
		errorBot: NewDingDingBot(errorBotAccessToken, errorBotSecrect, chLen),
	}

	go hook.logBot.Start()
	go hook.errorBot.Start()

	return hook
}

func (hook *DingDingBotHook) Fire(entry *logrus.Entry) error {
	msg := fmt.Sprintf("\n%s \n%s\n", entry.Time.Format("2006-01-02 15:04:05.000"), entry.Message)
	switch entry.Level {
	case logrus.PanicLevel:
	case logrus.FatalLevel:
	case logrus.ErrorLevel:
	case logrus.WarnLevel:
		hook.errorBot.MsgCh <- msg
	case logrus.InfoLevel:
		hook.logBot.MsgCh <- msg
	case logrus.DebugLevel:
	default:
		return nil
	}

	return nil
}

func (hook *DingDingBotHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
