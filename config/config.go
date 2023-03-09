package config

import (
	"io"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Cluster string `yaml:"cluster"`
}

func (c *Config) Load(file io.Reader) error {
	stream, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(stream, c)
}
