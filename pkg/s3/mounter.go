package s3

import (
	"fmt"
	"os"
	"os/exec"

	"k8s.io/klog/v2"
)

// Mounter interface
type Mounter interface {
	Stage(stagePath string) error
	Unstage(stagePath string) error
	Mount(source string, target string) error
}

// newMounter returns a new mounter
func newMounter(meta *metadata, cfg *Config) Mounter {
	return &s3fsMounter{
		metadata:      meta,
		url:           cfg.Endpoint,
		region:        cfg.Region,
		pwFileContent: cfg.AccessKeyID + ":" + cfg.SecretAccessKey,
	}
}

// Implements Mounter
type s3fsMounter struct {
	metadata      *metadata
	url           string
	region        string
	pwFileContent string
}

const (
	s3fsCmd = "s3fs"
)

func (s3fs *s3fsMounter) Stage(stageTarget string) error {
	return nil
}

func (s3fs *s3fsMounter) Unstage(stageTarget string) error {
	return nil
}

func (s3fs *s3fsMounter) Mount(source string, target string) error {
	if err := writes3fsPass(s3fs.pwFileContent); err != nil {
		return err
	}
	args := []string{
		fmt.Sprintf("%s:/%s", s3fs.metadata.Name, s3fs.metadata.FSPath),
		target,
		"-o", "use_path_request_style",
		"-o", fmt.Sprintf("url=%s", s3fs.url),
		"-o", fmt.Sprintf("endpoint=%s", s3fs.region),
		"-o", "allow_other",
		"-o", "mp_umask=000",
	}
	return fuseMount(s3fsCmd, args)
}

func writes3fsPass(pwFileContent string) error {
	pwFileName := fmt.Sprintf("%s/.passwd-s3fs", os.Getenv("HOME"))
	pwFile, err := os.OpenFile(pwFileName, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	_, err = pwFile.WriteString(pwFileContent)
	if err != nil {
		return err
	}
	pwFile.Close()
	return nil
}

func fuseMount(command string, args []string) error {
	cmd := exec.Command(command, args...)
	klog.Infof("mounting fuse with command:%s with args:%s", command, args)

	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("mounting fuse with command:%s with args:%s error:%s", command, args, string(out))
		return fmt.Errorf("fuseMount command:%s with args:%s error:%s", command, args, string(out))
	}

	return nil
}
