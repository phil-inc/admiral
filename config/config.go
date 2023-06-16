package config

import (
	"io"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Cluster  string    `yaml:"cluster"`
	Globals  globals   `yaml:"globals"`
	Watchers []watcher `yaml:"watchers"`
}

type globals struct {
	Backend backend `yaml:"backend"`
}

type watcher struct {
	Type                      string   `yaml:"type"`
	Backend                   backend  `yaml:"backend"`
	PodAnnotation             string   `yaml:"podAnnotation"`
	IgnoreContainerAnnotation string   `yaml:"ignoreContainerAnnotation"`
	Filter                    []string `yaml:"filter"`
}

type backend struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

func (c *Config) Load(file io.Reader) error {
	stream, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(stream, c)
}
