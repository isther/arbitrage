package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/isther/arbitrage/utils"
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
`
)

// ServerConfig defines the config of the server
type ServerConfig struct {
	UseProxy         bool   `json:"useProxy" yaml:"useProxy"`
	Proxy            string `json:"proxy" yaml:"proxy"`
	BinanceApiKey    string `json:"binanceApiKey" yaml:"binanceApiKey"`
	BinanceSecretKey string `json:"binanceSecretKey" yaml:"binanceSecretKey"`
	MexcApiKey       string `json:"mexcApiKey" yaml:"mexcApiKey"`
	MexcSecretKey    string `json:"mexcSecretKey" yaml:"mexcSecretKey"`
}

func loadConfigFile(filename string) error {
	if filename == "" {
		return errors.New("Config filename is empty")
	}

	if !utils.PathExists(filename) {
		utils.CreateAndWriteFile(filepath.Join(".", filename), configFileContent)
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
