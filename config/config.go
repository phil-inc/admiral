package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var (
	ConfigFileName = ".admiral.yaml"
)

type Config struct {
	Fargate   bool      `json:"fargate,omitempty"`
	Events    Events    `json:"events"`
	Logstream Logstream `json:"logstream"`
	Metrics   Metrics   `json:"metrics"`
	Namespace string    `json:"namespace,omitempty"`
	Cluster   string    `json:"cluster,omitempty"`
}

type Metrics struct {
	Handler MetricsHandler `json:"handler"`
	Apps    []string       `json:"apps"`
}

type MetricsHandler struct {
	Prometheus string `json:"prometheus"`
}

type Logstream struct {
	Logstore Logstore `json:"logstore"`
	Apps     []string `json:"apps"`
}

type Logstore struct {
	Loki Loki `json:"loki"`
}

type Loki struct {
	Url string `json:"url"`
}

type Events struct {
	Handler EventsHandler `json:"handler"`
}

type EventsHandler struct {
	Webhook Webhook `json:"webhook"`
}

type Webhook struct {
	Url string `json:"url"`
}

func New(path string) (*Config, error) {
	c := &Config{}
	if err := c.Load(path); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Config) Load(path string) error {
	if path == "" {
		path = getConfigFile()
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	a, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if len(a) != 0 {
		return yaml.Unmarshal(a, c)
	}

	return nil
}

func getConfigFile() string {
	configFile := filepath.Join(configDir(), ConfigFileName)
	if _, err := os.Stat(configFile); err == nil {
		return configFile
	}

	return ""
}

func configDir() string {
	if configDir := os.Getenv("CONFIG"); configDir != "" {
		return configDir
	}

	return os.Getenv("HOME")
}
