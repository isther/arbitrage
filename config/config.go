package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func Load(filename string) {
	// Load config
	loadConfigFile(filename)

	if Config.UseProxy {
		// Set env
		os.Setenv("HTTP_PROXY", Config.Addr)
		os.Setenv("HTTPS_PROXY", Config.Addr)
	}
}

var (
	Config            *ServerConfig
	configFileContent = `
proxy:
  useProxy: false
  proxy: ""
key:
  binanceApiKey: ""
  binanceSecretKey: ""
  mexcApiKey: ""
  mexcSecretKey: ""
  mexcCookie: ""
logDingDingConfig:
  accessToken: ""
  secrect: ""
errorDingDingConfig:
  accessToken: ""
  secrect: ""
params:
  number: 1
  minRatio: 0.7 # base 10000
  maxRatio: 1.5 # base 10000
  profitRatio: 0.1 # base 10000
  closeTimeOut: 1000 # ms
  klineRatio: 5.0 # base 10000
  klinePauseDuration: 500 # ms
  clientTimeOut: 500 # ms
  clientTimeOutPauseDuration: 500 # ms
  waitDuration: 5000 # ms
`
)

// ServerConfig defines the config of the server
type ServerConfig struct {
	Proxy               `json:"proxy" yaml:"proxy"`
	Key                 `json:"api" yaml:"api"`
	Params              `json:"params" yaml:"params"`
	LogDingDingConfig   DingDing `json:"logDingDingConfig" yaml:"logDingDingConfig"`
	ErrorDingDingConfig DingDing `json:"errorDingDingConfig" yaml:"errorDingDingConfig"`
}

type Proxy struct {
	UseProxy bool   `json:"useProxy" yaml:"useProxy"`
	Addr     string `json:"addr" yaml:"addr"`
}

type Key struct {
	BinanceApiKey    string `json:"binanceApiKey" yaml:"binanceApiKey"`
	BinanceSecretKey string `json:"binanceSecretKey" yaml:"binanceSecretKey"`
	MexcApiKey       string `json:"mexcApiKey" yaml:"mexcApiKey"`
	MexcSecretKey    string `json:"mexcSecretKey" yaml:"mexcSecretKey"`
	MexcCookie       string `json:"mexcCookie" yaml:"mexcCookie"`
}

type Params struct {
	Number                     int64   `json:"number" yaml:"number"`
	MinRatio                   float64 `json:"minRatio" yaml:"minRatio"`                                     // base 10000
	MaxRatio                   float64 `json:"maxRatio" yaml:"maxRatio"`                                     // base 10000
	ProfitRatio                float64 `json:"profitRatio" yaml:"profitRatio"`                               // base 10000
	CloseTimeOut               int64   `json:"closeTimeOut" yaml:"closeTimeOut"`                             // ms
	KlineRatio                 float64 `json:"klineRatio" yaml:"klineRatio"`                                 // base 10000
	KlinePauseDuration         int64   `json:"klinePauseDuration" yaml:"klinePauseDuration"`                 // ms
	ClientTimeOut              int64   `json:"clientTimeOut" yaml:"clientTimeOut"`                           // ms
	ClientTimeOutPauseDuration int64   `json:"clientTimeOutPauseDuration" yaml:"clientTimeOutPauseDuration"` // ms
	WaitDuration               int64   `json:"waitDuration" yaml:"waitDuration"`                             // ms
}

type DingDing struct {
	AccessToken string `json:"accessToken" yaml:"accessToken"`
	Secrect     string `json:"secrect" yaml:"secrect"`
}

func loadConfigFile(filename string) error {
	if filename == "" {
		return errors.New("Config filename is empty")
	}

	if !PathExists(filename) {
		CreateAndWriteFile(filepath.Join(".", filename), configFileContent)
	}

	contentBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file, err: %s", err.Error())
	}

	var sc ServerConfig
	if err := yaml.Unmarshal(contentBytes, &sc); err != nil {
		return fmt.Errorf("failed to unmarshal, err: %s", err.Error())
	}

	Config = &sc

	return nil
}
