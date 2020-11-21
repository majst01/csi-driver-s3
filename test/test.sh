#!/usr/bin/env bash
export MINIO_ACCESS_KEY=FJDSJ
export MINIO_SECRET_KEY=DSG643HGDS

mkdir -p /tmp/minio
minio server /tmp/minio &>/dev/null &
sleep 5
go test github.com/majst01/csi-driver-s3/pkg/s3 -cover
