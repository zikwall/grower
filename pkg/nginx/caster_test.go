package nginx

import (
	"testing"
	"time"
)

// nolint:gocyclo,funlen // cyclomatic complexity not important here
func TestCasterTryCast(t *testing.T) {
	t.Run("it should be successful cas String types", func(t *testing.T) {
		testCases := map[string]string{
			RemoteAddr:    "114.119.133.192",
			RemoteUser:    "test",
			Request:       "GET /sito/wp-includes/wlwmanifest.xml HTTP/1.1",
			HTTPReferer:   "empty",
			HTTPUserAgent: "User Agent Here",
			RequestMethod: "GET",
		}
		typeCaster := NewTypeCaster(&CasterCfg{})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			if receive != expect {
				t.Fatalf("failed, expect %s, receive %s", expect, receive)
			}
		}
	})
	t.Run("it should be successful cas Int32 types", func(t *testing.T) {
		testCases := map[string]string{
			ConnectionsWaiting: "190",
			ConnectionsActive:  "260",
			Connection:         "310",
			RequestLength:      "450",
		}
		expectedCases := map[string]int32{
			ConnectionsWaiting: 190,
			ConnectionsActive:  260,
			Connection:         310,
			RequestLength:      450,
		}
		typeCaster := NewTypeCaster(&CasterCfg{})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case int32:
			default:
				t.Fatalf("failed, expect %v, receive %v", "int32", it)
			}
			if receive != expectedCases[key] {
				t.Fatalf("failed, expect %d, receive %d", expectedCases[key], receive)
			}
		}
	})
	t.Run("it should be successful cas Float32 types", func(t *testing.T) {
		testCases := map[string]string{
			RequestTime:          "190.010",
			UpstreamConnectTime:  "260.010",
			UpstreamHeaderTime:   "310.010",
			UpstreamResponseTime: "450.010",
			MSec:                 "567.022",
		}
		expectedCases := map[string]float32{
			RequestTime:          190.010,
			UpstreamConnectTime:  260.010,
			UpstreamHeaderTime:   310.010,
			UpstreamResponseTime: 450.010,
			MSec:                 567.022,
		}
		typeCaster := NewTypeCaster(&CasterCfg{})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case float32:
			default:
				t.Fatalf("failed, expect %v, receive %v", "float32", it)
			}
			if receive != expectedCases[key] {
				t.Fatalf("failed, expect %f, receive %f", expectedCases[key], receive)
			}
		}
	})
	t.Run("it should be successful cas UInt32 types", func(t *testing.T) {
		testCases := map[string]string{
			BytesSent:     "190111222",
			BodyBytesSent: "260111222",
		}
		expectedCases := map[string]uint32{
			BytesSent:     190111222,
			BodyBytesSent: 260111222,
		}
		typeCaster := NewTypeCaster(&CasterCfg{})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case uint32:
			default:
				t.Fatalf("failed, expect %v, receive %v", "uint32", it)
			}
			if receive != expectedCases[key] {
				t.Fatalf("failed, expect %d, receive %d", expectedCases[key], receive)
			}
		}
	})
	t.Run("it should be successful cas Datetime types", func(t *testing.T) {
		var (
			t1 = time.Now()
			t2 = time.Now().Add(time.Second)
		)
		testCases := map[string]string{
			TimeLocal:   t1.Format("02/Jan/2006:15:04:05 -0700"),
			TimeISO8601: t2.Format(time.RFC3339),
		}
		expectedCases := map[string]time.Time{
			TimeLocal:   t1,
			TimeISO8601: t2,
		}
		typeCaster := NewTypeCaster(&CasterCfg{
			LocalTimeFormat: "02/Jan/2006:15:04:05 -0700",
		})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case time.Time:
				if !it.Truncate(time.Second).Equal(expectedCases[key].Truncate(time.Second)) {
					t.Fatalf("failed, expect %v, receive %v", expectedCases[key], receive)
				}
			default:
				t.Fatalf("failed, expect %v, receive %v", "time.Time", it)
			}
		}
	})
	t.Run("it should be successful cas Uint16 types", func(t *testing.T) {
		testCases := map[string]string{
			Status: "503",
		}
		expectedCases := map[string]uint16{
			Status: 503,
		}
		typeCaster := NewTypeCaster(&CasterCfg{})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case uint16:
			default:
				t.Fatalf("failed, expect %v, receive %v", "uint32", it)
			}
			if receive != expectedCases[key] {
				t.Fatalf("failed, expect %d, receive %d", expectedCases[key], receive)
			}
		}
	})
	t.Run("it should be successful cas Custom types", func(t *testing.T) {
		now := time.Now().UTC()
		testCases := map[string]string{
			"t1": "123456",
			"t2": "Some String",
			"t3": now.Format("2006-01-02T15:04:05"),
		}
		expectedCases := map[string]interface{}{
			"t1": int32(123456),
			"t2": "Some String",
			"t3": now,
		}
		typeCaster := NewTypeCaster(&CasterCfg{
			CustomCasts: map[string]string{
				"t1": "Integer",
				"t2": "String",
				"t3": "Datetime",
			},
			CustomCastsEnable: true,
		})
		for key, expect := range testCases {
			receive, err := typeCaster.TryCast(key, expect)
			if err != nil {
				t.Fatal(err)
			}
			switch it := receive.(type) {
			case time.Time:
				if !it.Truncate(time.Second).Equal(expectedCases[key].(time.Time).Truncate(time.Second)) {
					t.Fatalf("failed, expect %v, receive %v", expectedCases[key], receive)
				}
			case int32, string:
				if receive != expectedCases[key] {
					t.Fatalf("failed, expect %d, receive %d", expectedCases[key], receive)
				}
			default:
				t.Fatalf("failed, expect one of %s, receive %v", "int32, string, time.Time", it)
			}
		}
	})
}
