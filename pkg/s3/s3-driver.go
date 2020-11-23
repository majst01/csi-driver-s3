/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package s3

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"

	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/metal-stack/v"
)

type S3Driver struct {
	driver   *csicommon.CSIDriver
	endpoint string

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

type s3Volume struct {
	VolName string `json:"volName"`
	VolID   string `json:"volID"`
	VolSize int64  `json:"volSize"`
	VolPath string `json:"volPath"`
}

var (
	driverName = "s3.csi.metal-stack.io"
)

// New initializes the driver
func New(nodeID string, endpoint string) (*S3Driver, error) {
	driver := csicommon.NewCSIDriver(driverName, v.Version, nodeID)
	if driver == nil {
		klog.Fatalln("Failed to initialize CSI Driver.")
	}

	s3d := &S3Driver{
		endpoint: endpoint,
		driver:   driver,
	}
	return s3d, nil
}

func (s3 *S3Driver) newIdentityServer(d *csicommon.CSIDriver) *identityServer {
	return &identityServer{
		DefaultIdentityServer: csicommon.NewDefaultIdentityServer(d),
	}
}

func (s3 *S3Driver) newControllerServer(d *csicommon.CSIDriver) *controllerServer {
	return &controllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(d),
	}
}

func (s3 *S3Driver) newNodeServer(d *csicommon.CSIDriver) *nodeServer {
	return &nodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d),
	}
}

// Run the driver
func (s3 *S3Driver) Run() {
	klog.Infof("Driver: %v ", driverName)
	klog.Infof("Version: %v ", v.V)
	// Initialize default library driver

	s3.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME})
	s3.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	// Create GRPC servers
	s3.ids = s3.newIdentityServer(s3.driver)
	s3.ns = s3.newNodeServer(s3.driver)
	s3.cs = s3.newControllerServer(s3.driver)

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(s3.endpoint, s3.ids, s3.cs, s3.ns)
	s.Wait()
}
