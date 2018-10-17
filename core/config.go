package core

type Config struct {
	ConfigFilePath    string `json:"-"`
	DownloadDirectory string `json:"download_directory" default:"./downloads"`
	HttpServerPort    string `json:"http_server_port" default:"8080"`
}

func NewDefaultClientConfig() *Config {
	return &Config{
		ConfigFilePath: "./pooflix.json",
	}
}
