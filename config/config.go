package config

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Nginx  Nginx  `yaml:"nginx"`
	Scheme Scheme `yaml:"scheme"`
}

type Nginx struct {
	LogType              string            `yaml:"log_type"`
	LogTimeFormat        string            `yaml:"log_time_format"`
	LogFormat            string            `yaml:"log_format"`
	LogTimeRewrite       bool              `yaml:"log_time_rewrite"`
	LogCustomCastsEnable bool              `yaml:"log_custom_casts_enable"`
	LogCustomCasts       map[string]string `yaml:"log_custom_casts"`
}

type Scheme struct {
	Columns   map[string]string
	LogsTable string
}

func (s *Scheme) MapKeys() []string {
	keys := make([]string, 0, len(s.Columns))
	for key := range s.Columns {
		keys = append(keys, key)
	}
	return keys
}

func New(filepath string) (*Config, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}
