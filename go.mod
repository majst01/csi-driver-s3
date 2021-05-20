module github.com/majst01/csi-driver-s3

go 1.16

require (
	github.com/container-storage-interface/spec v1.4.0
	github.com/kubernetes-csi/csi-lib-utils v0.9.1 // indirect
	github.com/kubernetes-csi/csi-test v2.2.0+incompatible
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/metal-stack/v v1.0.3
	github.com/minio/minio-go/v7 v7.0.10
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	golang.org/x/net v0.0.0-20210428140749-89ef3d95e781
	google.golang.org/grpc v1.38.0
	k8s.io/klog/v2 v2.8.0
)
