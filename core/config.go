package core

import (
	"encoding/json"
	"errors"
	"github.com/creasty/defaults"
	"github.com/imdario/mergo"
	"os"
)

type Config struct {
	ConfigFilePath    string `json:"-"`
	DownloadDirectory string `json:"download_directory" default:"./downloads"`
	HttpServerPort    string `json:"http_server_port" default:"8080"`
}

func NewDefaultClientConfig() (*Config, error) {
	c := &Config{
		ConfigFilePath: "./pooflix.json",
	}

	if c.ConfigFilePath != "" {
		var configFileSettings Config
		configFile, err := os.Open(c.ConfigFilePath)
		if err != nil {
			return nil, err
		}

		if err := json.NewDecoder(configFile).Decode(&configFileSettings); err != nil {
			return nil, err
		}

		// Merge in command line settings (which overwrite respective config file settings)
		if err := mergo.Merge(c, configFileSettings); err != nil {
			return nil, err
		}

		// Set Default Settings with struct tags
		if err := defaults.Set(c); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("config is missing")
	}

	return c, nil
}
