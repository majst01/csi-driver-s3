package s3

import (
	"fmt"
	"os/exec"

	"k8s.io/klog/v2"
)

// Mounter interface which can be implemented
// by the different mounter types
type Mounter interface {
	Stage(stagePath string) error
	Unstage(stagePath string) error
	Mount(source string, target string) error
}

const (
	s3fsMounterType   = "s3fs"
	rcloneMounterType = "rclone"
	mounterTypeKey    = "mounter"
)

// newMounter returns a new mounter depending on the mounterType parameter
func newMounter(bucket *bucket, cfg *Config) (Mounter, error) {
	mounter := bucket.Mounter
	// Fall back to mounterType in cfg
	if len(bucket.Mounter) == 0 {
		mounter = cfg.Mounter
	}
	switch mounter {
	case s3fsMounterType:
		return newS3fsMounter(bucket, cfg)
	case rcloneMounterType:
		return newRcloneMounter(bucket, cfg)
	default:
		return newRcloneMounter(bucket, cfg)
	}
}

func fuseMount(path string, command string, args []string) error {
	cmd := exec.Command(command, args...)
	klog.Infof("Mounting fuse with command: %s and args: %s", command, args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error fuseMount command: %s\nargs: %s\noutput: %s", command, args, out)
	}

	return nil
}
