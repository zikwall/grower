package nginx

import (
	"errors"
	"strconv"
	"time"
)

const defaultDatetimeFormat = "2006-01-02T15:04:05"

type TypeCaster interface {
	TryCast(key, value string) (interface{}, error)
}

type CasterCfg struct {
	CustomCasts       map[string]string
	LocalTimeFormat   string
	CustomCastsEnable bool
	RemoveHyphen      bool
}

const (
	StringCustom   = "String"
	IntegerCustom  = "Integer"
	DatetimeCustom = "Datetime"
)

type caster struct {
	cfg        *CasterCfg
	hasCustoms bool
}

var (
	ErrCanNotParseTime    = errors.New("can't parse datetime string")
	ErrCanNotParseInt     = errors.New("can't parse integer value")
	ErrCanNotParseFloat32 = errors.New("can't parse float32 value")
)

func (c *caster) TryCast(key, value string) (interface{}, error) {
	if isHyphen(value) {
		value = ""
	}
	switch key {
	case TimeLocal, TimeISO8601:
		if value == "" {
			return time.Now(), nil
		}
		var layout string
		if key == TimeISO8601 {
			layout = time.RFC3339
		} else {
			layout = c.cfg.LocalTimeFormat
		}
		parsedTime, err := time.Parse(layout, value)
		if err != nil {
			return time.Now(), ErrCanNotParseTime
		}
		return parsedTime, nil
	case RemoteAddr, RemoteUser, Request, HTTPReferer, HTTPUserAgent, RequestMethod, HTTPS:
		return value, nil
	case BytesSent, BodyBytesSent, ConnectionsWaiting, ConnectionsActive, Status, Connection, RequestLength:
		if value == "" {
			return 0, nil
		}
		val, err := strconv.Atoi(value)
		if err != nil {
			return 0, ErrCanNotParseInt
		}
		return val, nil
	case RequestTime, UpstreamConnectTime, UpstreamHeaderTime, UpstreamResponseTime, MSec:
		if value == "" {
			return 0, nil
		}
		val, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return 0, ErrCanNotParseFloat32
		}
		return val, nil
	}
	if c.cfg.CustomCastsEnable && c.hasCustoms {
		if custom, ok := c.cfg.CustomCasts[key]; ok {
			switch custom {
			case StringCustom:
				return value, nil
			case IntegerCustom:
				if value == "" {
					return 0, nil
				}
				val, err := strconv.Atoi(value)
				if err != nil {
					return 0, ErrCanNotParseFloat32
				}
				return val, nil
			case DatetimeCustom:
				if value == "" {
					return time.Now(), nil
				}
				parsedTime, err := time.Parse(defaultDatetimeFormat, value)
				if err != nil {
					return parsedTime, ErrCanNotParseTime
				}
				return parsedTime, nil
			}
		}
	}
	return value, nil
}

var hyphenLen = len("-")

func isHyphen(value string) bool {
	if len(value) == hyphenLen && value == "-" {
		return true
	}
	return false
}

func NewTypeCaster(cfg *CasterCfg) TypeCaster {
	return &caster{
		cfg:        cfg,
		hasCustoms: len(cfg.CustomCasts) > 0,
	}
}
