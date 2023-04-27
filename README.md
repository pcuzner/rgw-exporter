# rgw-exporter
Prometheus exporter to provide RADOS gateway (RGW) user and bucket usage information

## Motivations
In older versions of Ceph, RGW bucket and user related information is not exposed to Prometheus, making reporting and 
alerting for Object based clusters problematic. However, object clusters can support 000's of users and millions of objects
so the exporter needs to attempt to account for large scale environments.

To address scale, the exporter is designed with the following characteristics

* utilizes [go-ceph](http://github.com/ceph/go-ceph) bindings (don't reinvent the wheel!)
* uses goroutines to gather bucket stats
* supports multiple rgw hosts, spreading API requests across them
* uses minimum thresholds for bucket size or object counts to limit the amount of metrics returned to Prometheus

By using the minimum thresholds, you can reduce the metrics that are returned and stored within Prometheus and focus only on the 'big hitters'. For example, if you only want to track bucket level stats for buckets over 10GiB in size, specify -threshold.size='10GiB'. 

Per user totals for objects and used capacity are calculated and returned to Prometheus, regardless of the thresholds.


## Running the Exporter 

Three environment variables need to be set;

| Variable | Description |
|----------|-------------|
| RGW_HOST | comma separated strings of the RGW hosts in the form ```http(s)://host:port,...``` |
| ACCESS_KEY | rgw admin ops access key |
| SECRET_KEY | rgw admin ops secret key |

Normally you'd expect to use an admin account with full privileges when using the admin API, but since the exporter doesn't need to change anything you can create a specific account with limited privileges, like this;

```
# radosgw-admin user create --uid=exporter --display-name="RGW exporter User"
# radosgw-admin caps add --uid=exporter --caps="metadata=read;usage=read;info=read;buckets=read;users=read"
```

### From the CLI

Once compiled, the exporter supports a number of flags;

```
# ./rgw-exporter -h 
  -debug
        run in debug mode (default true)
  -port int
        port for the exporter to bind to (default 9198)
  -threshold.objects uint
        minimum object count for per bucket reporting (default 1)
  -threshold.size string
        minimum bucket size for per bucket reporting (default "1Mb")
```

### Using a Container
You can use the `build-exporter.sh` script in the buildah folder to build the rgw-exporter container.

Once built, place your environment variables in a file and modify permissions to secure the credentials (there's also an example in the buildah folder of the format of this file). You can then run the container with something like this;

```
# podman run -d --network=host --env-file=rgw-exporter.env --name=rgw-exporter localhost/rgw-exporter:latest
```

Note, this example is using a local registry. For testing you can : ```podman pull docker.io/pcuzner/rgw-exporter:ff720512```

*The rgw-exporter container is approx. 24MB*

## Example Output
### Log
```
# ./rgw-exporter 
INFO[2023-04-22 16:30:47] Starting rgw-exporter                        
INFO[2023-04-22 16:30:47] RGW_HOST provides %d hosts1                  
INFO[2023-04-22 16:30:47] Parameters:                                  
INFO[2023-04-22 16:30:47] - RGW endpoints    : 1                       
INFO[2023-04-22 16:30:47] - Min object count : 1                       
INFO[2023-04-22 16:30:47] - Min bucket size  : 1000000 bytes (1Mb)     
INFO[2023-04-22 16:30:47] Binding to port 9198                         
DEBUG[2023-04-22 16:30:52] bucket list for uid dashboard using http://192.168.124.3:80 
DEBUG[2023-04-22 16:30:52] bucket list for uid admin using http://192.168.124.3:80 
DEBUG[2023-04-22 16:30:52] bucket list for uid exporter using http://192.168.124.3:80 
DEBUG[2023-04-22 16:30:52] Waiting for go routines to finish            
DEBUG[2023-04-22 16:30:52] bucket stats for user 'exporter' starting    
DEBUG[2023-04-22 16:30:52] bucket stats for user 'dashboard' starting   
DEBUG[2023-04-22 16:30:52] bucket stats for user 'admin' starting       
DEBUG[2023-04-22 16:30:52] bucket stats for user 'dashboard' complete   
DEBUG[2023-04-22 16:30:52] bucket stats for user 'exporter' complete    
DEBUG[2023-04-22 16:30:52] bucket stats for user 'admin' complete       
DEBUG[2023-04-22 16:30:52] go routines completed in : 41.974825ms       
INFO[2023-04-22 16:30:52] Processing the bucket stats data             
DEBUG[2023-04-22 16:30:52] Processing buckets for user: exporter        
INFO[2023-04-22 16:30:52] Processing complete  
```

### Metrics
```
# HELP ceph_rgw_bucket_object_count Count of objects stored in a bucket
# TYPE ceph_rgw_bucket_object_count gauge
ceph_rgw_bucket_object_count{bucket="mytest",uid="exporter"} 1
# HELP ceph_rgw_bucket_shard_count The number of RADOS objects(shards) a bucket index is using
# TYPE ceph_rgw_bucket_shard_count gauge
ceph_rgw_bucket_shard_count{bucket="mytest",uid="exporter"} 11
# HELP ceph_rgw_bucket_usage_bytes Total data stored in a bucket
# TYPE ceph_rgw_bucket_usage_bytes gauge
ceph_rgw_bucket_usage_bytes{bucket="mytest",uid="exporter"} 1.048576e+06
# HELP ceph_rgw_user RGW user
# TYPE ceph_rgw_user gauge
ceph_rgw_user{uid="admin"} 1
ceph_rgw_user{uid="dashboard"} 1
ceph_rgw_user{uid="exporter"} 1
# HELP ceph_rgw_user_total_bucket_count Total number of buckets owned by the user
# TYPE ceph_rgw_user_total_bucket_count gauge
ceph_rgw_user_total_bucket_count{uid="admin"} 0
ceph_rgw_user_total_bucket_count{uid="dashboard"} 0
ceph_rgw_user_total_bucket_count{uid="exporter"} 1
# HELP ceph_rgw_user_total_usage_bytes Total of stored data for a given user
# TYPE ceph_rgw_user_total_usage_bytes gauge
ceph_rgw_user_total_usage_bytes{uid="admin"} 0
ceph_rgw_user_total_usage_bytes{uid="dashboard"} 0
ceph_rgw_user_total_usage_bytes{uid="exporter"} 1.048576e+06
```