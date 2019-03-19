package publicLog

import "testing"

func TestNewOfflineFileLog(t *testing.T) {
	publicLogger, err := NewOfflineFileLog("logTestDir/", "helloTest")
	if err != nil {
		panic(err)
	}
	publicLogger.LogPublic("test", map[string]interface{}{
		"driver_id":          123,
		"pid":                23333,
		"did":                3192541,
		"appversion":         "v1.0.2",
		"cancel_reason_type": 30,
		"hello":              "world",
	})
}
