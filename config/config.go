package config

import (
	"io"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Cluster string `yaml:"cluster"`
	Logs logs `yaml:"logs"`
	Events events `yaml:"events"`
}

type logs struct {
	Namespace string `yaml:"namespace"`
	Backend backend `yaml:"backend"`
	PodAnnotation string `yaml:"podAnnotation"`
	IgnoreContainerAnnotation string `yaml:"ignoreContainerAnnotation"`
}

type events struct {
	Namespace string `yaml:"namespace"`
	Backend backend `yaml:"backend"`
	Filter []string `yaml:"filter"`
}

type backend struct {
	Type string `yaml:"type"`
	URL string `yaml:"url"`
}

func (c *Config) Load(file io.Reader) error {
	stream, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(stream, c)
}
