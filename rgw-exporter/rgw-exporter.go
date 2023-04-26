// rgw-exporter is a metrics exporter for Prometheus environments. It is intended
// to provide bucket and user level information to help with monitoring and alerting
// in Ceph RadosGW clusters.
package main

import (
	"flag"
	"fmt"

	// "log"
	"net/http"
	"os"
	"strings"

	"github.com/pcuzner/rgw-exporter/collector"
	"github.com/pcuzner/rgw-exporter/defaults"
	"github.com/pcuzner/rgw-exporter/utils"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
	})
}

func main() {
	var accessKey, secretKey, hostString string
	var config defaults.Config

	// flag returns a pointer, not a value..
	port := flag.Int("port", defaults.DefaultPort, "port for the exporter to bind to")
	debug := flag.Bool("debug", true, "run in debug mode")
	thresholdSize := flag.String("threshold.size", defaults.MinBucketSize, "minimum bucket size for per bucket reporting")
	thresholdObjects := flag.Uint64("threshold.objects", defaults.MinObjectCount, "minimum object count for per bucket reporting")

	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.Info("Starting rgw-exporter")

	minSize, err := utils.TextToBytes(*thresholdSize)
	if err != nil {
		log.Fatalf("Invalid threshold.size('%s'): %s. Aborting", *thresholdSize, err)
	}

	hostString, hostOK := os.LookupEnv("RGW_HOST")
	accessKey, accessOK := os.LookupEnv("ACCESS_KEY")
	secretKey, secretOK := os.LookupEnv("SECRET_KEY")

	if !hostOK || !accessOK || !secretOK {
		log.Fatal("One of more environment variables missing. Requires: RGW_HOST, ACCESS_KEY and SECRET_KEY")
	}

	candidateHosts := strings.Split(hostString, ",")
	log.Info("RGW_HOST provides %d hosts", len(candidateHosts))
	hosts, err := utils.ValidateHosts(candidateHosts)
	if err != nil {
		log.Fatalln("No valid endpoints provided. Aborting")
	}

	config.Endpoints = hosts
	config.AccessKey = accessKey
	config.SecretKey = secretKey
	config.ThresholdObjects = *thresholdObjects
	config.ThresholdSize = minSize

	log.Info("Parameters:")
	log.Infof("- RGW endpoints    : %d", len(hosts))
	log.Infof("- Min object count : %d", *thresholdObjects)
	log.Infof("- Min bucket size  : %d bytes (%s)", minSize, *thresholdSize)

	rgwCollector := collector.NewRGWCollector(&config)
	prometheus.MustRegister(rgwCollector)

	log.Infof("Binding to port %d", *port)
	http.Handle("/metrics", promhttp.Handler())

	listenAddr := fmt.Sprintf(":%d", *port)
	log.Fatal(http.ListenAndServe(listenAddr, nil))

}
