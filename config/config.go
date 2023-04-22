package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/isther/arbitrage/utils"
	"gopkg.in/yaml.v2"
)

var (
	Config            *ServerConfig
	configFileContent = `apiKey: ""
secretKey: ""
`
)

// ServerConfig defines the config of the server
type ServerConfig struct {
	Proxy     string `json:"proxy" yaml:"proxy"`
	ApiKey    string `json:"apiKey" yaml:"apiKey"`
	SecretKey string `json:"secretKey" yaml:"secretKey"`
}

func LoadConfig(filename string) error {
	if filename == "" {
		return errors.New("use -c to specify configuration file")
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
