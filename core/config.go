package core

type Config struct {
	ConfigFilePath    string `json:"-"`
	DownloadDirectory string `json:"download_directory" default:"./downloads"`
}

func NewDefaultClientConfig() *Config {
	return &Config{
		ConfigFilePath: "./pooflix.json",
	}
}
