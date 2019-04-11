// Mounter executes mount/umount commands, find out mount list

package mounter

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"                //TODO use interface?
	k8sMount "k8s.io/kubernetes/pkg/util/mount" //TODO use interface?
)

// Mounter executes mount/umount commands, finds out mount list
type Mounter struct {
	log   *logrus.Entry
	mount k8sMount.Interface
}

// New creates a new Mounter
func New(log *logrus.Entry) *Mounter {
	l := log.WithField("cmp", "Mounter")

	return &Mounter{
		log:   l,
		mount: k8sMount.New(""),
	}
}

// FindMountByTargetPath finds mount by mount target path
func (m *Mounter) FindMountByTargetPath(targetPath string) (*k8sMount.MountPoint, error) {
	allMounts, err := m.mount.List()
	if err != nil {
		return nil, fmt.Errorf("InternalError: Cannot get shares list: %s", err)
	}

	for _, mount := range allMounts {
		if mount.Path == targetPath {
			return &mount, nil
		}
	}

	return nil, nil
}

// FindMountBySource finds mounts by mount source
func (m *Mounter) FindMountBySource(mountSource string) ([]k8sMount.MountPoint, error) {
	mounts := []k8sMount.MountPoint{}

	allMounts, err := m.mount.List()
	if err != nil {
		return nil, fmt.Errorf("InternalError: Cannot get shares list: %s", err)
	}

	for _, mount := range allMounts {
		if mount.Device == mountSource {
			mounts = append(mounts, mount)
		}
	}

	return mounts, nil
}

// BindMount prepares and executes bind mount command
func (m *Mounter) BindMount(mountSource, targetPath string) error {
	return m.Mount(mountSource, targetPath, "", []string{"bind", "remount"})
}

// Mount prepares and executes mount command
func (m *Mounter) Mount(mountSource, targetPath, fsType string, mountOptions []string) error {
	// check if mountpoint exists, create if there is no such directory
	notMountPoint, err := m.mount.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return fmt.Errorf(
					"InternalError: Failed to mkdir to share target path '%s': %s",
					targetPath,
					err,
				)
			}
			notMountPoint = true
		} else {
			return fmt.Errorf(
				"InternalError: Cannot ensure that target path '%s' can be used as a mount point: %s",
				targetPath,
				err,
			)
		}
	}

	if !notMountPoint { // already mounted
		return fmt.Errorf("InternalError: Target path '%s' is already a mount point", targetPath)
	}

	o := strings.Join(mountOptions, ",")
	if fsType == "" {
		m.log.Infof("mount -o %s %s %s", o, mountSource, targetPath)
	} else {
		m.log.Infof("mount -t %s -o %s %s %s", fsType, o, mountSource, targetPath)
	}

	err = m.mount.Mount(mountSource, targetPath, fsType, mountOptions)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf(
				"PermissionDenied: Permission denied to mount '%s' to '%s': %s",
				mountSource,
				targetPath,
				err,
			)
		} else if strings.Contains(err.Error(), "invalid argument") {
			return fmt.Errorf(
				"InvalidArgument: Cannot mount '%s' to '%s', invalid argument: %s",
				mountSource,
				targetPath,
				err,
			)
		}
		return fmt.Errorf(
			"InternalError: Failed to mount '%s' to '%s': %s",
			mountSource,
			targetPath,
			err,
		)
	}

	return nil
}

// DoUnmount prepares end executes umount command and removes mount point
func (m *Mounter) DoUnmount(targetPath string) error {
	if err := m.mount.Unmount(targetPath); err != nil {
		return fmt.Errorf("InternalError: Failed to unmount target path '%s': %s", targetPath, err)
	}

	notMountPoint, err := m.mount.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf(
			"InternalError: Cannot ensure that target path '%s' is a mount point: '%s'",
			targetPath,
			err,
		)
	} else if !notMountPoint { // still mounted
		return fmt.Errorf("InternalError: Target path '%s' is still mounted", targetPath)
	}

	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("InternalError: Cannot remove unmounted target path '%s': %s", targetPath, err)
	}

	return nil
}
