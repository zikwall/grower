package nginx

import (
	"bytes"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/zikwall/grower/config"
)

// nolint:lll // it's OK
const (
	caseOne = `114.119.133.192 - - [21/Jul/2022:00:30:43 +0300] "GET /sito/wp-includes/wlwmanifest.xml HTTP/1.1" 444 9 100000.14 "GET" "-" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36" ON 10 <2022-07-21T00:30:43> 8 16 32 64 | 11 22 33 44 | 1000 2000 | 1234567890_abcdefg | 2022-07-21`
)

var (
	caseOneTime, _       = time.Parse("02/Jan/2006:15:04:05 -0700", "21/Jul/2022:00:30:43 +0300")
	caseOneCustomTime, _ = time.Parse("2006-01-02T15:04:05", "2022-07-21T00:30:43")
	caseOneDate, _       = time.Parse("2006-01-02", "2022-07-21")
)

// nolint:lll // it's OK
var cases = map[string]map[string]interface{}{
	caseOne: {
		"remote_addr":        "114.119.133.192",
		"remote_user":        "",
		"time_local":         caseOneTime,
		"request":            "GET /sito/wp-includes/wlwmanifest.xml HTTP/1.1",
		"status":             uint16(444),
		"bytes_sent":         uint32(9),
		"request_time":       float32(100000.14),
		"request_method":     "GET",
		"http_referer":       "",
		"http_user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		"https":              "ON",
		"custom_field":       int32(10),
		"custom_time_field":  caseOneCustomTime,
		"field_uint8":        uint8(8),
		"field_uint16":       uint16(16),
		"field_uint32":       uint32(32),
		"field_uint64":       uint64(64),
		"field_int8":         int8(11),
		"field_int16":        int16(22),
		"field_int32":        int32(33),
		"field_int64":        int64(44),
		"field_f32":          float32(1000),
		"field_f64":          float64(2000),
		"field_fixed_string": "1234567890",
		"field_date":         caseOneDate,
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
				switch it := castedValue.(type) {
				case time.Time:
					if !it.Truncate(time.Second).Equal(expectedValue.(time.Time).Truncate(time.Second)) {
						t.Fatalf("failed for key %s, expect %v, receive %v", key, expectedValue, castedValue)
					}
				default:
					if castedValue != expectedValue {
						t.Fatalf("failed for key %s, expect %v, receive %v", key, expectedValue, castedValue)
					}
				}
			}
		}
	})
}
