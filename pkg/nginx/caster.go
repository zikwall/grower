package nginx

import (
	"errors"
	"strconv"
	"time"
)

const (
	defaultDatetimeFormat = "2006-01-02T15:04:05"
	defaultDateFormat     = "2006-01-02"
)

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

// Clickhouse native types
const (
	UInt8       = "UInt8"
	UInt16      = "UInt16"
	UInt32      = "UInt32"
	UInt64      = "UInt64"
	Int8        = "Int8"
	Int16       = "Int16"
	Int32       = "Int32"
	Int64       = "Int64"
	String      = StringCustom
	FixedString = "FixedString"
	Float32     = "Float32"
	Float64     = "Float64"
	Date        = "Date"
	DateTime    = "DateTime"
)

var FixedStringPrefixLen = len("FixedString")

type caster struct {
	cfg        *CasterCfg
	hasCustoms bool
}

var (
	ErrCanNotParseTime    = errors.New("can't parse datetime/date string")
	ErrCanNotParseUInt8   = errors.New("can't parse uint8 value")
	ErrCanNotParseUInt16  = errors.New("can't parse uint16 value")
	ErrCanNotParseUInt32  = errors.New("can't parse uint32 value")
	ErrCanNotParseUInt64  = errors.New("can't parse uint64 value")
	ErrCanNotParseInt8    = errors.New("can't parse int8 value")
	ErrCanNotParseInt16   = errors.New("can't parse int16 value")
	ErrCanNotParseInt32   = errors.New("can't parse int32 value")
	ErrCanNotParseInt64   = errors.New("can't parse int64 value")
	ErrCanNotParseFloat32 = errors.New("can't parse float32 value")
	ErrCanNotParseFloat64 = errors.New("can't parse float64 value")
	ErrCanNotParseFixedSz = errors.New("can't parse fixed string size")
)

// nolint:gocyclo // it's ok
func (c *caster) TryCast(key, value string) (interface{}, error) {
	if isHyphen(value) {
		value = ""
	}
	// this block can rewrite the standard attributes of Nginx itself,
	// if it is necessary to make their own conversion for them.
	if c.cfg.CustomCastsEnable && c.hasCustoms {
		if custom, ok := c.cfg.CustomCasts[key]; ok {
			switch custom {
			case UInt8:
				return parseUInt8(value)
			case UInt16:
				return parseUInt16(value)
			case UInt32:
				return parseUInt32(value)
			case UInt64:
				return parseUInt64(value)
			case Int8:
				return parseInt8(value)
			case Int16:
				return parseInt16(value)
			case Int32, IntegerCustom:
				return parseInt32(value)
			case Int64:
				return parseInt64(value)
			case Float32:
				return parseFloat32(value)
			case Float64:
				return parseFloat64(value)
			case String:
				return value, nil
			case Date:
				return parseDateTime(value, defaultDateFormat)
			case DateTime, DatetimeCustom:
				return parseDateTime(value, defaultDatetimeFormat)
			}
			// special case for clickhouse FixedString type
			if isFixedString(custom) && value != "" {
				return unsafeParseFixedStringN(custom, value)
			}
		}
	}
	return c.nnv(key, value)
}

// nnv - nginx native value
func (c *caster) nnv(key, value string) (interface{}, error) {
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
	return value, nil
}

var hyphenLen = len("-")

func isHyphen(value string) bool {
	if len(value) == hyphenLen && value == "-" {
		return true
	}
	return false
}

var (
	bracketA uint8 = '('
	bracketB uint8 = ')'
)

func isFixedString(key string) bool {
	return len(key) > FixedStringPrefixLen && key[:FixedStringPrefixLen] == FixedString
}

// unsafeParseFixedStringN return first N characters from string
// unsafeParseFixedStringN unsafe for direct use
// unsafeParseFixedStringN use with isFixedString function
func unsafeParseFixedStringN(key, value string) (string, error) {
	key = key[FixedStringPrefixLen:]
	if len(key) <= 2 {
		return "", nil
	}
	if key[0] != bracketA || key[len(key)-1] != bracketB {
		return "", nil
	}
	size, err := parseInt32(key[1 : len(key)-1])
	if err != nil {
		return "", ErrCanNotParseFixedSz
	}
	if len(value) <= int(size) {
		return value, nil
	}
	return value[:size], nil
}

// generated functions

func parseUInt8(value string) (uint8, error) {
	if value == "" {
		return uint8(0), nil
	}
	val, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return uint8(0), ErrCanNotParseUInt8
	}
	return uint8(val), nil
}

func parseUInt16(value string) (uint16, error) {
	if value == "" {
		return uint16(0), nil
	}
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

func parseUInt64(value string) (uint64, error) {
	if value == "" {
		return uint64(0), nil
	}
	val, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return uint64(0), ErrCanNotParseUInt64
	}
	return val, nil
}

func parseInt8(value string) (int8, error) {
	if value == "" {
		return int8(0), nil
	}
	val, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return int8(0), ErrCanNotParseInt8
	}
	return int8(val), nil
}

func parseInt16(value string) (int16, error) {
	if value == "" {
		return int16(0), nil
	}
	val, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return int16(0), ErrCanNotParseInt16
	}
	return int16(val), nil
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

func parseInt64(value string) (int64, error) {
	if value == "" {
		return int64(0), nil
	}
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return int64(0), ErrCanNotParseInt64
	}
	return val, nil
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

func parseFloat64(value string) (float64, error) {
	if value == "" {
		return float64(0), nil
	}
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return float64(0), ErrCanNotParseFloat64
	}
	return val, nil
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
