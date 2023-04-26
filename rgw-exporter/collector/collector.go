// collector package handles all the functions related to gathering the data
// from RGW and marshalling into Prometheus metrics
package collector

import (
	"context"
	"time"

	// "log"
	"sync"

	"github.com/pcuzner/rgw-exporter/defaults"

	log "github.com/Sirupsen/logrus"

	rgw "github.com/ceph/go-ceph/rgw/admin"
	"github.com/prometheus/client_golang/prometheus"
)

type rgwCollector struct {
	config         defaults.Config
	rgwUser        *prometheus.Desc
	bucketSize     *prometheus.Desc
	bucketObjects  *prometheus.Desc
	numShards      *prometheus.Desc
	summaryBuckets *prometheus.Desc
	summaryUsage   *prometheus.Desc
}

// UserStats holds the summary data for a given users S3 usage
type UserStats struct {
	bucketCount float64
	totalBytes  float64
}

// NewRGWCollector creates a new collector containing the the runtime
// configuration and the prometheus metrics that the collector will return
func NewRGWCollector(config *defaults.Config) *rgwCollector {
	return &rgwCollector{
		config: *config,
		rgwUser: prometheus.NewDesc("ceph_rgw_user",
			"RGW user",
			[]string{"uid"}, nil,
		),
		bucketSize: prometheus.NewDesc("ceph_rgw_bucket_usage_bytes",
			"Total data stored in a bucket",
			[]string{"uid", "bucket"}, nil,
		),
		bucketObjects: prometheus.NewDesc("ceph_rgw_bucket_object_count",
			"Count of objects stored in a bucket",
			[]string{"uid", "bucket"}, nil,
		),
		numShards: prometheus.NewDesc("ceph_rgw_bucket_shard_count",
			"The number of RADOS objects(shards) a bucket index is using",
			[]string{"uid", "bucket"}, nil,
		),
		summaryBuckets: prometheus.NewDesc("ceph_rgw_user_total_bucket_count",
			"Total number of buckets owned by the user",
			[]string{"uid"}, nil,
		),
		summaryUsage: prometheus.NewDesc("ceph_rgw_user_total_usage_bytes",
			"Total of stored data for a given user",
			[]string{"uid"}, nil,
		),
	}
}

// Describe returns the metric metadata
func (collector *rgwCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.rgwUser
	ch <- collector.bucketSize
	ch <- collector.bucketObjects
	ch <- collector.numShards
	ch <- collector.summaryBuckets
	ch <- collector.summaryUsage
}

// Collect handles the data gathering from RGW and pushes the metrics
// onto the channel for return to the Prometheus server
func (collector *rgwCollector) Collect(ch chan<- prometheus.Metric) {

	connection, err := getRGWConnection(collector.config.Endpoints, 0, collector.config.AccessKey, collector.config.SecretKey)
	if err != nil {
		log.Fatalln("Unable to connect to any endpoint. Aborting")
	}

	users := collectUsers(connection)
	var wg sync.WaitGroup
	queue := make(chan []rgw.Bucket, len(users))
	wg.Add(len(users))

	userSummary := make(map[string]UserStats, len(users))

	for _, uid := range users {
		userSummary[uid] = UserStats{0, 0}
		metric := prometheus.MustNewConstMetric(collector.rgwUser, prometheus.GaugeValue, 1, uid)
		ch <- metric
	}
	start := time.Now()
	hostIdx := 0
	for _, uid := range users {
		conn, err := getRGWConnection(collector.config.Endpoints, hostIdx, collector.config.AccessKey, collector.config.SecretKey)
		if err != nil {
			log.Fatalln("Unable to connect to any endpoint. Aborting")
		}
		log.Debug("bucket list for uid ", uid, " using ", conn.Endpoint)
		go collectBucketStats(conn, uid, &wg, queue)
		hostIdx++
		if hostIdx > len(collector.config.Endpoints)-1 {
			hostIdx = 0
		}
	}

	log.Debug("Waiting for go routines to finish")
	wg.Wait()
	elapsed := time.Since(start)
	log.Debugf("go routines completed in : %s", elapsed)

	close(queue)

	log.Info("Processing the bucket stats data")
	for bucketData := range queue {
		if len(bucketData) == 0 {
			continue
		}
		for _, bucketInfo := range bucketData {
			var sizeFlag, objectFlag bool = false, false

			log.Debug("Processing buckets for user: ", bucketInfo.Owner)
			var entry UserStats = userSummary[bucketInfo.Owner]

			if bucketInfo.Usage.RgwMain.Size != nil {
				if *bucketInfo.Usage.RgwMain.Size >= collector.config.ThresholdSize {
					metric := prometheus.MustNewConstMetric(
						collector.bucketSize, prometheus.GaugeValue,
						float64(*bucketInfo.Usage.RgwMain.Size),
						bucketInfo.Owner, bucketInfo.Bucket)
					ch <- metric
					sizeFlag = true
				}

				entry.totalBytes += float64(*bucketInfo.Usage.RgwMain.Size)

			}
			if bucketInfo.Usage.RgwMain.NumObjects != nil {
				if *bucketInfo.Usage.RgwMain.NumObjects >= collector.config.ThresholdObjects {
					metric := prometheus.MustNewConstMetric(
						collector.bucketObjects, prometheus.GaugeValue,
						float64(*bucketInfo.Usage.RgwMain.NumObjects),
						bucketInfo.Owner, bucketInfo.Bucket)
					ch <- metric
					objectFlag = true
				}
			}

			if sizeFlag || objectFlag {
				metric := prometheus.MustNewConstMetric(
					collector.numShards,
					prometheus.GaugeValue,
					float64(*bucketInfo.NumShards),
					bucketInfo.Owner, bucketInfo.Bucket)
				ch <- metric
			}

			entry.bucketCount += 1
			userSummary[bucketInfo.Owner] = entry
		}

	}
	for uid, summaryStats := range userSummary {
		totalBuckets := prometheus.MustNewConstMetric(
			collector.summaryBuckets, prometheus.GaugeValue,
			summaryStats.bucketCount,
			uid)
		ch <- totalBuckets

		totalUsage := prometheus.MustNewConstMetric(
			collector.summaryUsage, prometheus.GaugeValue,
			summaryStats.totalBytes,
			uid)
		ch <- totalUsage
	}
	log.Info("Processing complete")
}

// getRGWConnection returns an API connection to an RGW gateway
// It receives the list of potenital hosts and an offset to try. If a connection fails it
// will continue to the next host until all hosts are exhausted - at which point it returns an error
func getRGWConnection(hosts []string, offset int, accessKey string, secretKey string) (*rgw.API, error) {
	var failures int = 0
	var conn *rgw.API
	var err error
	var targets []string

	if offset == 0 {
		targets = hosts
	} else {
		targets = append(targets, hosts[offset:]...)
		targets = append(targets, hosts[0:(offset-1)]...)
	}

	for _, hostName := range targets {
		conn, err = rgw.New(hostName, accessKey, secretKey, nil)
		if err == nil {
			break
		}
		log.Warning("Unable to connect to ", hostName)
		failures++

	}
	if failures == len(hosts) {
		log.Error("All hosts given have been tried and are NOT reachable.")
	}
	return conn, err
}

// collectUsers uses the rgw API to return a list of RGW users
func collectUsers(connection *rgw.API) []string {
	users, err := connection.GetUsers(context.Background())
	if err != nil {
		// TODO - don't fail if there are no users
		log.Fatal("Unable to get a user list: ", err)
	}
	return *users
}

// collectBucketStats returns the bucket stats for all buckets owned by a given user
func collectBucketStats(connection *rgw.API, uid string, wg *sync.WaitGroup, ch chan<- []rgw.Bucket) {
	log.Debugf("bucket stats for user '%s' starting", uid)
	userBuckets, err := connection.ListUsersBucketsWithStat(context.Background(), uid)
	if err != nil {
		log.Warning("Warning: Unable to list buckets for uid ", uid, " : ", err)
	}
	log.Debugf("bucket stats for user '%s' complete", uid)
	ch <- userBuckets
	defer wg.Done()
}
