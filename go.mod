module github.com/majst01/csi-driver-s3

go 1.16

require (
	github.com/container-storage-interface/spec v1.5.0
	github.com/kubernetes-csi/csi-lib-utils v0.10.0 // indirect
	github.com/kubernetes-csi/csi-test v2.2.0+incompatible
	github.com/kubernetes-csi/drivers v1.0.2
	github.com/metal-stack/v v1.0.3
	github.com/minio/minio-go/v7 v7.0.12
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d
	google.golang.org/grpc v1.40.0
	k8s.io/klog/v2 v2.10.0
)
