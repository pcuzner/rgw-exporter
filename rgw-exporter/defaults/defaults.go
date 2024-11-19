// defaults defines the default settings and structs used by the rgw-exporter
package defaults

// Default Listening port
var DefaultPort int = 9198

// Object count threshold for per bucket metrics
var MinObjectCount uint64 = 1

// Bucket size (used capacity) threshold for per bucket reporting
// this is a human readable string like 1GiB
var MinBucketSize string = "1Mb"

// Config holds the key attributes that the collector needs to run
type Config struct {
	Endpoints        []string
	AccessKey        string
	SecretKey        string
	ThresholdSize    uint64
	ThresholdObjects uint64
	SkipTLSVerify    bool
}
