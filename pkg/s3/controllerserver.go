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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
}

func (cs *controllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	s3, _ := newS3ClientFromSecrets(req.GetSecrets())
	bucketName := *s3.bucketName
	volumeID := req.GetName()

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		klog.Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	if len(bucketName) != 0 {
		volumeID = bucketName
	} else if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Name missing in request")
	} else {
		return nil, status.Error(codes.InvalidArgument, "Bucket Name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}

	capacityBytes := int64(req.GetCapacityRange().GetRequiredBytes())

	klog.Infof("Got a request to create volume %s", volumeID)

	err := ensureBucketWithMetadata(volumeID, req.GetSecrets(), capacityBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create backup and metadata:%v", err)
	}
	klog.Infof("create volume %s", volumeID)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: capacityBytes,
			VolumeContext: req.GetParameters(),
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	s3, err := newS3ClientFromSecrets(req.GetSecrets())
	bucketName := *s3.bucketName
	volumeID := req.GetVolumeId()

	// Check arguments
	if len(bucketName) != 0 {
		volumeID = bucketName
	} else if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	} else {
		return nil, status.Error(codes.InvalidArgument, "Bucket Name missing in request")
	}

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		klog.Errorf("Invalid delete volume req: %v", req)
		return nil, err
	}
	klog.Infof("Deleting volume %s", volumeID)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}
	exists, err := s3.bucketExists(volumeID)
	if err != nil {
		return nil, err
	}
	if exists {
		if err := s3.removeBucket(volumeID); err != nil {
			klog.Errorf("Failed to remove volume %s: %w", volumeID, err)
			return nil, err
		}
	} else {
		klog.Infof("Bucket %s does not exist, ignoring request", volumeID)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}

	s3, err := newS3ClientFromSecrets(req.GetSecrets())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}
	exists, err := s3.bucketExists(req.GetVolumeId())
	if err != nil {
		return nil, err
	}
	if !exists {
		// return an error if the volume requested does not exist
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Volume with id %s does not exist", req.GetVolumeId()))
	}

	// We currently only support RWO
	supportedAccessMode := &csi.VolumeCapability_AccessMode{
		Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}

	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != supportedAccessMode.GetMode() {
			return &csi.ValidateVolumeCapabilitiesResponse{Message: "Only single node writer is supported"}, nil
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessMode: supportedAccessMode,
				},
			},
		},
	}, nil
}

func (cs *controllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return &csi.ControllerExpandVolumeResponse{}, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")
}

func sanitizeVolumeID(volumeID string) string {
	volumeID = strings.ToLower(volumeID)
	if len(volumeID) > 63 {
		h := sha256.New()
		_, _ = io.WriteString(h, volumeID)
		volumeID = hex.EncodeToString(h.Sum(nil))
	}
	return volumeID
}

func ensureBucketWithMetadata(volumeID string, secrets map[string]string, capacityBytes int64) error {
	s3, err := newS3ClientFromSecrets(secrets)
	if err != nil {
		return fmt.Errorf("failed to initialize S3 client: %w", err)
	}
	exists, err := s3.bucketExists(volumeID)
	if err != nil {
		return fmt.Errorf("failed to check if bucket %s exists: %w", volumeID, err)
	}
	if exists {
		var meta *metadata
		if !s3.metadataExist(volumeID) {
			b := &metadata{
				Name:          volumeID,
				CapacityBytes: capacityBytes,
				FSPath:        fsPrefix,
			}
			if err := s3.writeMetadata(b); err != nil {
				return fmt.Errorf("Error setting volume metadata: %w", err)
			}
		}
		meta, err = s3.getMetadata(volumeID)
		if err != nil {
			return fmt.Errorf("failed to get metadata of volume %s: %w", volumeID, err)
		}
		// Check if volume capacity requested is bigger than the already existing capacity
		if capacityBytes > meta.CapacityBytes {
			return status.Error(codes.AlreadyExists, fmt.Sprintf("Volume with the same name: %s but with smaller size already exist", volumeID))
		}
	} else {
		if err = s3.createBucket(volumeID); err != nil {
			return fmt.Errorf("failed to create bucket for volume %s: %w", volumeID, err)
		}
		if err = s3.createPrefix(volumeID, fsPrefix); err != nil {
			return fmt.Errorf("failed to create prefix %s for volume %s: %w", fsPrefix, volumeID, err)
		}
		meta := &metadata{
			Name:          volumeID,
			CapacityBytes: capacityBytes,
			FSPath:        fsPrefix,
		}
		if err := s3.writeMetadata(meta); err != nil {
			return fmt.Errorf("Error setting volume metadata: %w", err)
		}
	}
	return nil
}
