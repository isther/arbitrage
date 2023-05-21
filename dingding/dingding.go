package dingding

import (
	"github.com/CatchZeng/dingtalk/pkg/dingtalk"
	"github.com/sirupsen/logrus"
)

var ()

func hh() {
	logrus.AddHook(&DingDingBotHook{})
}

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
	switch entry.Level {
	case logrus.PanicLevel:
	case logrus.FatalLevel:
	case logrus.ErrorLevel:
	case logrus.WarnLevel:
		hook.errorBot.MsgCh <- dingtalk.NewTextMessage().SetContent(entry.Message)
	case logrus.InfoLevel:
		hook.logBot.MsgCh <- dingtalk.NewTextMessage().SetContent(entry.Message)
	case logrus.DebugLevel:
	default:
		return nil
	}

	return nil
}

func (hook *DingDingBotHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
