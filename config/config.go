package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Nginx  Nginx  `yaml:"nginx"`
	Scheme Scheme `yaml:"scheme"`
}

type Nginx struct {
	LogCustomCasts       map[string]string `yaml:"log_custom_casts"`
	LogType              string            `yaml:"log_type"`
	LogTimeFormat        string            `yaml:"log_time_format"`
	LogFormat            string            `yaml:"log_format"`
	LogTimeRewrite       bool              `yaml:"log_time_rewrite"`
	LogCustomCastsEnable bool              `yaml:"log_custom_casts_enable"`
	LogRemoveHyphen      bool              `yaml:"log_remove_hyphen"`
}

type Scheme struct {
	Columns   map[string]string `yaml:"columns"`
	LogsTable string            `yaml:"logs_table"`
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
	if config.Scheme.LogsTable == "" {
		return nil, fmt.Errorf("logs table is not provided")
	}
	if len(config.Scheme.Columns) == 0 {
		return nil, fmt.Errorf("table schema is empty")
	}
	if config.Nginx.LogFormat == "" {
		return nil, fmt.Errorf("log format is empty")
	}
	return config, nil
}
