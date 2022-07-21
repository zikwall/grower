package config

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
	Columns map[string]string
}
