#!/usr/bin/bash

# Requires buildah and podman

VERSION="1.0"
SHORTID=$(git rev-parse HEAD | head -c 8)
IMAGE_TAG="${VERSION}-${SHORTID}"
BUILD_REPO="${REPO:=docker.io}"

if [ ! -z "$1" ]; then
  IMAGE_TAG=$1
fi

read -p "The build will attempt to push to ${BUILD_REPO}, so press ENTER when you're logged in."
echo "Build image with the tag: $IMAGE_TAG"

# podman pull the image first, push to your local registry to speed up local builds
#IMAGE="docker.io/golang:1.19"
IMAGE="${BUILD_REPO}/golang:1.19"
build=$(buildah from $IMAGE)
buildah add $build ../rgw-exporter /rgw-exporter
buildah config --workingdir /rgw-exporter $build
buildah run -e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=0 -- $build go build .

# as above, grab the alpine image first and push to a local registry
#container=$(buildah from "docker.io/alpine:3.17")
container=$(buildah from "${BUILD_REPO}/alpine:3.17")
buildah config --workingdir / $container
buildah copy --from $build $container /rgw-exporter/rgw-exporter /rgw-exporter
buildah config --entrypoint "/rgw-exporter" $container

buildah config --label maintainer="Paul Cuzner <pcuzner@ibm.com>" $container
buildah config --label description="Ceph radosgw exporter" $container
buildah config --label summary="Ceph radosgw exporter" $container
buildah commit --format docker --squash $container rgw-exporter:$IMAGE_TAG

if [ "${BUILD_REPO}" == "docker.io" ]; then
  echo -e "\nTagging the image"
  podman tag localhost/rgw-exporter:${IMAGE_TAG} ${BUILD_REPO}/pcuzner/rgw-exporter:${IMAGE_TAG}

  echo -e "\nPushing the image to ${BUILD_REPO}"
  podman push ${BUILD_REPO}/pcuzner/rgw-exporter:${IMAGE_TAG}
fi
