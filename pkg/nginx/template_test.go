package nginx

import (
	"bytes"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/zikwall/ck-nginx/config"
)

// nolint:lll // it's OK
const (
	caseOne = `114.119.133.192 - - [21/Jul/2022:00:30:43 +0300] "GET /sito/wp-includes/wlwmanifest.xml HTTP/1.1" 444 9 100000.14 "GET" "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36" ON 10 <2022-07-21T00:30:43>`
)

var (
	caseOneTime, _       = time.Parse("02/Jan/2006:15:04:05 -0700", "21/Jul/2022:00:30:43 +0300")
	caseOneCustomTime, _ = time.Parse("2006-01-02T15:04:05", "2022-07-21T00:30:43")
)

// nolint:lll // it's OK
var cases = map[string]map[string]interface{}{
	caseOne: {
		"remote_addr":       "114.119.133.192",
		"remote_user":       "",
		"time_local":        caseOneTime,
		"request":           "GET /sito/wp-includes/wlwmanifest.xml HTTP/1.1",
		"status":            uint16(444),
		"bytes_sent":        uint32(9),
		"request_time":      float32(100000.14),
		"request_method":    "GET",
		"http_referer":      "",
		"http_user_agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		"https":             "ON",
		"custom_field":      int32(10),
		"custom_time_field": caseOneCustomTime,
	},
}

func TestTemplate(t *testing.T) {
	var (
		cfg        = &config.Config{}
		template   *Template
		typeCaster TypeCaster
	)
	t.Run("it should be successfully prepare tests", func(t *testing.T) {
		content, err := os.ReadFile("../../sample_test.yaml")
		if err != nil {
			t.Fatal(err)
		}
		decoder := yaml.NewDecoder(bytes.NewReader(content))
		if err := decoder.Decode(&cfg); err != nil {
			t.Fatal(err)
		}
		template = NewTemplate(cfg.Nginx.LogFormat)
		typeCaster = NewTypeCaster(&CasterCfg{
			CustomCasts:       cfg.Nginx.LogCustomCasts,
			LocalTimeFormat:   cfg.Nginx.LogTimeFormat,
			CustomCastsEnable: cfg.Nginx.LogCustomCastsEnable,
			RemoveHyphen:      false,
		})
	})
	t.Run("it should be successfully get values", func(t *testing.T) {
		for cas, entries := range cases {
			entry, err := template.ParseString(cas)
			if err != nil {
				t.Fatal(err)
			}
			for key, expectedValue := range entries {
				field, err := entry.Field(key)
				if err != nil {
					t.Fatal(err)
				}
				castedValue, err := typeCaster.TryCast(key, field)
				if err != nil {
					t.Fatal(err)
				}
				if castedValue != expectedValue {
					t.Fatalf("failed for key %s, expect %v, receive %v", key, expectedValue, castedValue)
				}
			}
		}
	})
}
