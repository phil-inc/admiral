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

type Handler struct {
	Webhook Webhook `json:"webhook"`
}

type Config struct {
	Events    Events    `json:"events"`
	Logstream Logstream `json:"logstream"`
	Namespace string    `json:"namespace,omitempty"`
	Cluster   string    `json:"cluster,omitempty"`
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
	Handler Handler `json:"handler"`
}

type Webhook struct {
	Url string `json:"url"`
}

func New() (*Config, error) {
	c := &Config{}
	if err := c.Load(); err != nil {
		return c, err
	}

	return c, nil
}

func createIfNotExist() error {
	configFile := filepath.Join(configDir(), ConfigFileName)
	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(configFile)
			if err != nil {
				return err
			}
			file.Close()
		} else {
			return err
		}
	}
	return nil
}

func (c *Config) Load() error {
	err := createIfNotExist()
	if err != nil {
		return err
	}

	file, err := os.Open(getConfigFile())
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
