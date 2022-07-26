package nginx

// TimeLocal, TimeISO8601 Time constants
const (
	TimeLocal   = "time_local"
	TimeISO8601 = "time_iso8601"
)

// String constants
const (
	RemoteAddr    = "remote_addr"
	RemoteUser    = "remote_user"
	Request       = "request"
	HTTPReferer   = "http_referer"
	HTTPUserAgent = "http_user_agent"
	RequestMethod = "request_method"
	HTTPS         = "https"
)

// Integer constants
const (
	ConnectionsWaiting = "connections_waiting"
	ConnectionsActive  = "connections_active"
	Connection         = "connection"
	RequestLength      = "request_length"
)

// Unsigned Integer32 constants
const (
	BytesSent     = "bytes_sent"
	BodyBytesSent = "body_bytes_sent"
)

// Status Unsigned Integer16 constant
const Status = "status"

// Float32 constants
const (
	RequestTime          = "request_time"
	UpstreamConnectTime  = "upstream_connect_time"
	UpstreamHeaderTime   = "upstream_header_time"
	UpstreamResponseTime = "upstream_response_time"
	MSec                 = "msec"
)
