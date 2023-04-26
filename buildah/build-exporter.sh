#!/usr/bin/bash

# Requires buildah and podman

if [ ! -z "$1" ]; then
  TAG=$1
else
  TAG='latest'
fi

echo "Build image with the tag: $TAG"

# podman pull the image first, push to your local registry to speed up local builds
#IMAGE="docker.io/golang:1.19"
IMAGE="laptop:5000/golang:1.19"
build=$(buildah from $IMAGE)
buildah add $build ../rgw-exporter /rgw-exporter
buildah config --workingdir /rgw-exporter $build
buildah run -e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=0 -- $build go build .

# as above, grab the alpine image first and push to a local registry
#container=$(buildah from "docker.io/alpine:3.17")
container=$(buildah from "laptop:5000/alpine:3.17")
buildah config --workingdir / $container
buildah copy --from $build $container /rgw-exporter/rgw-exporter /rgw-exporter
buildah config --entrypoint "/rgw-exporter" $container

buildah config --label maintainer="Paul Cuzner <pcuzner@ibm.com>" $container
buildah config --label description="Ceph radosgw exporter" $container
buildah config --label summary="Ceph radosgw exporter" $container
buildah commit --format docker --squash $container rgw-exporter:$TAG