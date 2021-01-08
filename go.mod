module github.com/majst01/csi-driver-s3

go 1.15

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/kubernetes-csi/csi-lib-utils v0.9.0 // indirect
	github.com/kubernetes-csi/csi-test v2.2.0+incompatible
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/metal-stack/v v1.0.2
	github.com/minio/minio-go/v7 v7.0.7
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/grpc v1.34.0
	k8s.io/klog/v2 v2.4.0
)
