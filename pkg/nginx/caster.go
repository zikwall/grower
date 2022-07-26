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
	ErrCanNotParseInt32   = errors.New("can't parse int32 value")
	ErrCanNotParseUInt16  = errors.New("can't parse uint16 value")
	ErrCanNotParseUInt32  = errors.New("can't parse uint32 value")
	ErrCanNotParseFloat32 = errors.New("can't parse float32 value")
)

func (c *caster) TryCast(key, value string) (interface{}, error) {
	if isHyphen(value) {
		value = ""
	}
	switch key {
	case TimeLocal, TimeISO8601:
		var layout string
		if key == TimeISO8601 {
			layout = time.RFC3339
		} else {
			layout = c.cfg.LocalTimeFormat
		}
		return parseDateTime(value, layout)
	case Status:
		return parseUInt16(value)
	case BytesSent, BodyBytesSent:
		return parseUInt32(value)
	case RemoteAddr, RemoteUser, Request, HTTPReferer, HTTPUserAgent, RequestMethod, HTTPS:
		return value, nil
	case ConnectionsWaiting, ConnectionsActive, Connection, RequestLength:
		return parseInt32(value)
	case RequestTime, UpstreamConnectTime, UpstreamHeaderTime, UpstreamResponseTime, MSec:
		return parseFloat32(value)
	}
	if c.cfg.CustomCastsEnable && c.hasCustoms {
		if custom, ok := c.cfg.CustomCasts[key]; ok {
			switch custom {
			case StringCustom:
				return value, nil
			case IntegerCustom:
				return parseInt32(value)
			case DatetimeCustom:
				return parseDateTime(value, defaultDatetimeFormat)
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

func parseUInt16(value string) (uint16, error) {
	val, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return uint16(0), ErrCanNotParseUInt16
	}
	return uint16(val), nil
}

func parseUInt32(value string) (uint32, error) {
	if value == "" {
		return uint32(0), nil
	}
	val, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return uint32(0), ErrCanNotParseUInt32
	}
	return uint32(val), nil
}

func parseInt32(value string) (int32, error) {
	if value == "" {
		return int32(0), nil
	}
	val, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return int32(0), ErrCanNotParseInt32
	}
	return int32(val), nil
}

func parseFloat32(value string) (float32, error) {
	if value == "" {
		return float32(0), nil
	}
	val, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return float32(0), ErrCanNotParseFloat32
	}
	return float32(val), nil
}

func parseDateTime(value, format string) (time.Time, error) {
	if value == "" {
		return time.Now(), nil
	}
	parsedTime, err := time.Parse(format, value)
	if err != nil {
		return parsedTime, ErrCanNotParseTime
	}
	return parsedTime, nil
}

func NewTypeCaster(cfg *CasterCfg) TypeCaster {
	return &caster{
		cfg:        cfg,
		hasCustoms: len(cfg.CustomCasts) > 0,
	}
}
