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
		os.Setenv("HTTP_PROXY", Config.Proxy)
		os.Setenv("HTTPS_PROXY", Config.Proxy)
	}
}

var (
	Config            *ServerConfig
	configFileContent = `useProxy: false
proxy: ""
binanceApiKey: ""
binanceSecretKey: ""
mexcApiKey: ""
mexcSecretKey: ""
mexcCookie:""
minRatio: 0.0,
maxRatio: 0.0
profitRatio: 0.0
closeTimeOut: 500
number: 1
logDingDingConfig:
  accessToken: ""
  secrect: ""
errorDingDingConfig:
  accessToken: ""
  secrect: ""
`
)

// ServerConfig defines the config of the server
type ServerConfig struct {
	UseProxy            bool     `json:"useProxy" yaml:"useProxy"`
	Proxy               string   `json:"proxy" yaml:"proxy"`
	BinanceApiKey       string   `json:"binanceApiKey" yaml:"binanceApiKey"`
	BinanceSecretKey    string   `json:"binanceSecretKey" yaml:"binanceSecretKey"`
	MexcApiKey          string   `json:"mexcApiKey" yaml:"mexcApiKey"`
	MexcSecretKey       string   `json:"mexcSecretKey" yaml:"mexcSecretKey"`
	MexcCookie          string   `json:"mexcCookie" yaml:"mexcCookie"`
	MinRatio            float64  `json:"minRatio" yaml:"minRatio"`
	MaxRatio            float64  `json:"maxRatio" yaml:"maxRatio"`
	ProfitRatio         float64  `json:"profitRatio" yaml:"profitRatio"`
	CloseTimeOut        int64    `json:"closeTimeOut" yaml:"closeTimeOut"`
	Number              int64    `json:"number" yaml:"number"`
	LogDingDingConfig   DingDing `json:"logDingDingConfig" yaml:"logDingDingConfig"`
	ErrorDingDingConfig DingDing `json:"errorDingDingConfig" yaml:"errorDingDingConfig"`
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
