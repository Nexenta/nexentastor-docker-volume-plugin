package driver

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"

	"github.com/Nexenta/nexenta-docker-driver/pkg/arrays"
	"github.com/Nexenta/nexenta-docker-driver/pkg/config"
	"github.com/Nexenta/nexenta-docker-driver/pkg/mounter"
	"github.com/Nexenta/nexentastor-csi-driver/pkg/ns" //TODO move to a dedicated library
)

// Driver - Docker Volume Driver for NS, it implements methods /VolumeDriver.*:
// https://docs.docker.com/v17.09/engine/extend/plugins_volume/
type Driver struct {
	log        *logrus.Entry
	config     *config.Config
	nsResolver *ns.Resolver
	mounter    *mounter.Mounter
}

// Args - params to create a new driver
type Args struct {
	Config *config.Config
	Log    *logrus.Entry
}

// New - create new NS volume driver
func New(args Args) (*Driver, error) {
	l := args.Log.WithField("cmp", "Driver")
	l.Debug("created...")

	nsResolver, err := ns.NewResolver(ns.ResolverArgs{
		Address:  args.Config.Address,
		Username: args.Config.Username,
		Password: args.Config.Password,
		Log:      l,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot create NexentaStor resolver: %s", err)
	}

	return &Driver{
		log:        l,
		config:     args.Config,
		nsResolver: nsResolver,
		mounter:    mounter.New(l),
	}, nil
}

// refreshConfig reads config file and re-creates NS resolver if the config has been changed
func (d *Driver) refreshConfig() error {
	changed, err := d.config.Refresh()
	if err != nil {
		return err
	}

	if changed {
		d.nsResolver, err = ns.NewResolver(ns.ResolverArgs{
			Address:  d.config.Address,
			Username: d.config.Username,
			Password: d.config.Password,
			Log:      d.log,
		})
		if err != nil {
			return fmt.Errorf("Cannot create NexentaStor resolver: %s", err)
		}
	}

	return nil
}

// resolveNS finds NS to use by dataset or filesystem path
func (d *Driver) resolveNS(datasetPath string) (ns.ProviderInterface, error) {
	nsProvider, err := d.nsResolver.Resolve(datasetPath)
	if err != nil {
		humanizedErr := fmt.Errorf("Cannot resolve '%s' on any NexentaStor(s): %s", datasetPath, err)

		// propagate NefError
		nefCode := ns.GetNefErrorCode(err)
		if nefCode != "" { // TODO add IsNefError() method
			return nil, &ns.NefError{
				Err:  humanizedErr,
				Code: nefCode,
			}
		}

		return nil, humanizedErr
	}
	return nsProvider, nil
}

// Capabilities returns plugin capabilities
func (d *Driver) Capabilities() *volume.CapabilitiesResponse {
	l := d.log.WithField("func", "Capabilities()")
	l.Info("request")

	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "global",
		},
	}
}

// Create Docker volume, created filesystem on NS
func (d *Driver) Create(req *volume.CreateRequest) error {
	l := d.log.WithField("func", "Create()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	if err := d.refreshConfig(); err != nil {
		return logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	datasetPath := d.config.DefaultDataset
	filesystemPath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(datasetPath)
	if err != nil {
		return logError(l, err)
	}
	l.Infof("path '%s' resolved on %s NexentaStor", datasetPath, nsProvider)

	err = nsProvider.CreateFilesystem(ns.CreateFilesystemParams{Path: filesystemPath})
	if err != nil {
		if ns.IsAlreadyExistNefError(err) {
			l.Infof(
				"done: NexentaStor filesystem '%s' already exists and can be used for '%s' volume",
				filesystemPath,
				volumeName,
			)
			return nil
		}
		return logError(l, fmt.Errorf(
			"InternalError: Cannot create NexentaStor filesystem '%s' for volume '%s': %s",
			filesystemPath,
			volumeName,
			err,
		))
	}

	l.Infof("done: filesystem '%s' has been created on NexentaStore for '%s' volume", filesystemPath, volumeName)
	return nil
}

// Remove Docker volume, removes filesystem on NS
func (d *Driver) Remove(req *volume.RemoveRequest) error {
	l := d.log.WithField("func", "Remove()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	if err := d.refreshConfig(); err != nil {
		return logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	datasetPath := d.config.DefaultDataset
	filesystemPath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(filesystemPath)
	if err != nil {
		if ns.IsNotExistNefError(err) {
			l.Infof("done: NexentaStor filesystem '%v' already doesn't exist, return OK response", filesystemPath)
			return nil
		}
		return logError(l, err)
	}
	l.Infof("path '%s' resolved on %s NexentaStor", filesystemPath, nsProvider)

	// if here, than filesystemPath exists on NS
	err = nsProvider.DestroyFilesystemWithClones(filesystemPath, false)
	if err != nil && !ns.IsNotExistNefError(err) {
		return logError(l, fmt.Errorf(
			"Cannot delete NexentaStor filesystem '%s' for volume '%s': %s",
			filesystemPath,
			volumeName,
			err,
		))
	}

	l.Infof("done: filesystem '%s' has been deleted from NexentaStor for '%s' volume", filesystemPath, volumeName)
	return nil
}

// List volumes managed by NS
func (d *Driver) List() (*volume.ListResponse, error) {
	l := d.log.WithField("func", "List()")
	l.Infof("request")

	if err := d.refreshConfig(); err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	// a root of all driver's filesystem
	datasetPath := d.config.DefaultDataset

	nsProvider, err := d.resolveNS(datasetPath)
	if err != nil {
		return nil, logError(l, err)
	}
	l.Infof("path '%s' resolved on %s NexentaStor", datasetPath, nsProvider)

	filesystems, err := nsProvider.GetFilesystems(datasetPath)
	if err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot get filesystems: %s", err))
	}

	volumes := make([]*volume.Volume, len(filesystems))
	for i, item := range filesystems {
		name := strings.TrimPrefix(item.Path, datasetPath+"/")
		volumes[i] = &volume.Volume{
			Name: name,
			//Mountpoint: filepath.Join(config.DriverMountPointsRoot, name),
		}
	}

	l.Infof("done: found %d entries(s)", len(volumes))
	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

// Get volume by its name, find out if NS has this filesystem created
func (d *Driver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	l := d.log.WithField("func", "Get()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return nil, logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	if err := d.refreshConfig(); err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	datasetPath := d.config.DefaultDataset
	filesystemPath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(filesystemPath)
	if err != nil {
		if ns.IsNotExistNefError(err) {
			l.Infof("done: filesystem '%v' doesn't exist on NexentaStor, return empty response", filesystemPath)
			return nil, nil
		}
		return nil, logError(l, err)
	}
	l.Infof("path '%s' resolved on %s NexentaStor", datasetPath, nsProvider)

	filesystem, err := nsProvider.GetFilesystem(filesystemPath)
	if err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot get filesystem '%s': %s", filesystemPath, err))
	}

	l.Infof("done: filesystem '%s' was found for '%v' volume", filesystem.String(), volumeName)
	return &volume.GetResponse{
		Volume: &volume.Volume{
			Name: volumeName,
			//Mountpoint: filepath.Join(config.DriverMountPointsRoot, volumeName),
		},
	}, nil
}

// Path returns volume mount point
func (d *Driver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	l := d.log.WithField("func", "Path()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return nil, logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	// TODO this is volume mount point, not a container bind-mount point
	mountPoint := filepath.Join(config.DriverMountPointsRoot, volumeName)

	l.Infof("done: return mount point '%s' for '%v' volume", mountPoint, volumeName)
	return &volume.PathResponse{
		// as docs says (https://docs.docker.com/v17.09/engine/extend/plugins_volume/#volumedriverpath)
		// it's OK to return empty response, in our case driver use mount + bind-mounts for each container
		// and there is no way to say what is "Mountpoint" for particular Docker volume
		//Mountpoint: mountPoint,
	}, nil
}

// Mount mounts NS share to a Docker host, then bind-mounts this share to another folder for particular container.
//
// Working model:
// NS share "S" <---> Docker host folder "H" (mount -t nfs S H) <--+--> Mount for container A (mount -o bind H A)
//                                                                 |--> Mount for container B (mount -o bind H B)
//                                                                 |--> Mount for container C (mount -o bind H C)
//                                                                 `--> Mount for container D (mount -o bind H D)
//
// On host all 'mount' happen under:
// /var/lib/docker/plugins/<PLUGIN_ID>/propagated-mount/volume/<VOLUME_NAME>              - mounted NS share
// /var/lib/docker/plugins/<PLUGIN_ID>/propagated-mount/bind/<VOLUME_NAME>-<CONTAINER_ID> - bind container(s) to share
//
// Inside driver's container all 'mount' happen under:
// /mnt/nvd/volume/<VOLUME_NAME>              - mounted NS share
// /mnt/nvd/bind/<VOLUME_NAME>-<CONTAINER_ID> - bind container(s) to share
// `/mnt/nvd` is a "propagatedmount" parameter in the `config.json`.
//
func (d *Driver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	l := d.log.WithField("func", "Mount()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return nil, logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	containerID := req.ID
	if containerID == "" {
		return nil, logError(l, fmt.Errorf("InvalidArgument: req.ID must be provided"))
	}

	if err := d.refreshConfig(); err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	datasetPath := d.config.DefaultDataset
	filesystemPath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(filesystemPath)
	if err != nil {
		return nil, logError(l, err)
	}
	l.Infof("path '%s' resolved on %s NexentaStor", filesystemPath, nsProvider)

	// get NexentaStor filesystem information
	filesystem, err := nsProvider.GetFilesystem(filesystemPath)
	if err != nil {
		return nil, logError(l, fmt.Errorf("FailedPrecondition: Cannot get filesystem '%s': %s", filesystemPath, err))
	}

	// check if NS filesystem is shared over NFS, create NFS share if it doesn't exist
	if !filesystem.SharedOverNfs {
		err := d.createNfsShare(nsProvider, filesystem)
		if err != nil {
			return nil, logError(l, err)
		}
	}

	dataIP := d.config.DefaultDataIP
	volumeMountPoint := getVolumeMountPoint(volumeName) // path inside driver's container to mount NS filesystem

	//TODO get mount options from runtime params, set default if not specified?
	mountOptions := []string{}
	for _, option := range strings.Split(d.config.DefaultMountOptions, ",") {
		if option != "" {
			mountOptions = append(mountOptions, option)
		}
	}

	// mount filesystem to volume mount point
	err = d.mountNFSShare(filesystem, dataIP, volumeMountPoint, mountOptions)
	if err != nil {
		return nil, logError(l, err)
	}

	l.Infof(
		"filesystem share '%s' has been mounted to '%s'",
		getNFSMountSource(dataIP, filesystem.MountPoint),
		volumeMountPoint,
	)

	// bind mount volume mount to a container specific mount
	containerBindMountPoint := getContainerBindMountPath(volumeName, containerID)
	err = d.mounter.BindMount(volumeMountPoint, containerBindMountPoint)
	if err != nil {
		return nil, logError(l, err)
	}

	l.Infof(
		"done: volume mount point '%s' has been bind-mounted to container mount point '%s'",
		volumeMountPoint,
		containerBindMountPoint,
	)
	return &volume.MountResponse{
		Mountpoint: containerBindMountPoint,
	}, nil
}

func (d *Driver) mountNFSShare(filesystem ns.Filesystem, dataIP, targetPath string, mountOptions []string) error {
	// NFS style mount source
	mountSource := getNFSMountSource(dataIP, filesystem.MountPoint)

	// NFS v3 is used by default if no version specified by user
	mountOptions = arrays.AppendIfRegexpNotExistString(mountOptions, regexp.MustCompile("^vers=.*$"), "vers=3")

	// check if this filesystem is already mounted on the host
	// validate if this mount can be used within another container (has same source, target and options)
	existingMount, err := d.mounter.FindMountByTargetPath(targetPath)
	if err != nil {
		return err
	} else if existingMount != nil {
		d.log.Debugf("existing mount found: %+v", existingMount)

		if existingMount.Device != mountSource {
			return fmt.Errorf(
				"Mount point '%s' already exists and cannot be used for a new container, because mount sources are "+
					"different. Needed: '%s', already mounted: '%s'",
				targetPath,
				mountSource,
				existingMount.Device,
			)
		}

		// compare mount options
		missedOptions := []string{}
		for _, o := range mountOptions {
			if !arrays.ContainsString(existingMount.Opts, o) {
				missedOptions = append(missedOptions, o)
			}
		}
		if len(missedOptions) != 0 {
			return fmt.Errorf(
				"Mount '%s' (source: '%s') already exists, but cannot be used within the new container, "+
					"following mount options are missed: %v",
				targetPath,
				mountSource,
				missedOptions,
			)
		}

		d.log.Infof(
			"mount point '%s' (source: '%s') already exists and can be used within the new container",
			targetPath,
			mountSource,
		)
		return nil
	}

	return d.mounter.Mount(mountSource, targetPath, config.FsTypeNFS, mountOptions)
}

// createNfsShare creates filesystem share on NS, sets up ACL for it
func (d *Driver) createNfsShare(nsProvider ns.ProviderInterface, filesystem ns.Filesystem) error {
	err := nsProvider.CreateNfsShare(ns.CreateNfsShareParams{
		Filesystem: filesystem.Path,
	})
	if err != nil {
		return fmt.Errorf("InternalError: Cannot share filesystem '%s' over NFS: %s", filesystem.Path, err)
	}

	// TODO select read-only or read-write mount options set based on runtime parameters
	var aclRuleSet ns.ACLRuleSet
	aclRuleSet = ns.ACLReadWrite
	// if req.GetReadonly() {
	// 	aclRuleSet = ns.ACLReadOnly
	// } else {
	// 	aclRuleSet = ns.ACLReadWrite
	// }

	// apply NS filesystem ACL (gets applied only for new shares, not for already shared filesystems)
	err = nsProvider.SetFilesystemACL(filesystem.Path, aclRuleSet)
	if err != nil {
		return fmt.Errorf("InternalError: Cannot set filesystem ACL for '%s': %s", filesystem.Path, err)
	}

	return nil
}

// Unmount un-mounts container bind-mount and also un-mounts NS filesystem mount if no one is using it
func (d *Driver) Unmount(req *volume.UnmountRequest) error {
	l := d.log.WithField("func", "Unmount()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if volumeName == "" {
		return logError(l, fmt.Errorf("InvalidArgument: req.Name must be provided"))
	}

	containerID := req.ID
	if containerID == "" {
		return logError(l, fmt.Errorf("InvalidArgument: req.ID must be provided"))
	}

	if err := d.refreshConfig(); err != nil {
		return logError(l, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err))
	}

	dataIP := d.config.DefaultDataIP
	datasetPath := d.config.DefaultDataset
	filesystemPath := filepath.Join(datasetPath, volumeName)
	containerBindMountPoint := getContainerBindMountPath(volumeName, containerID)

	// unmount volume to container bind-mount
	err := d.mounter.DoUnmount(containerBindMountPoint)
	if err != nil {
		return logError(l, err)
	}
	l.Infof("container bind-mount '%s' has been unmounted", containerBindMountPoint)

	// NFS style mount source
	volumeMountSource := getNFSMountSource(dataIP, filesystemPath)

	nsProvider, err := d.resolveNS(filesystemPath)
	if err == nil {
		// get NexentaStor filesystem information
		filesystem, err := nsProvider.GetFilesystem(filesystemPath)
		if err == nil {
			// if filesystem found on NS, then update volume mount source
			// by actual share path from NS filesystem properties
			volumeMountSource = getNFSMountSource(dataIP, filesystem.MountPoint)
		}
	}
	l.Infof("volume mount point source to look for: %v", volumeMountSource)

	// check if volume mount is mounted including bind mount to containers
	volumeMounts, err := d.mounter.FindMountBySource(volumeMountSource)
	if err != nil {
		return logError(l, err)
	}
	l.Infof("found %d mount(s) that has '%s' source", len(volumeMounts), volumeMountSource)

	volumeMountPoint := getVolumeMountPoint(volumeName) // path inside driver's container to mount NS filesystem

	mountCount := len(volumeMounts)
	if mountCount == 0 {
		l.Infof("done: no mounts found with source '%s'", volumeMountSource)
	} else if mountCount == 1 {
		// this is the last filesystem share mount, therefore no container uses it, filesystem can be finally unmount
		l.Infof("the last filesystem mount with source '%s' was found, attempt to unmount it", volumeMountSource)
		err := d.mounter.DoUnmount(volumeMountPoint)
		if err != nil {
			return logError(l, err)
		}
		l.Infof("done: volume '%s' has been unmounted", volumeMountPoint)
	} else {
		l.Infof(
			"done: keep filesystem mount '%s -> %s' because it is used by %d other container(s)",
			volumeMountSource,
			volumeMountPoint,
			mountCount-1,
		)
	}

	return nil
}

// getVolumeMountPoint is a path inside driver's container:
// /mnt/nvd/volume/<VOLUME_NAME>
func getVolumeMountPoint(volumeName string) string {
	return filepath.Join(config.DriverMountPointsRoot, "volume", volumeName)
}

// getContainerBindMountPath is a path inside driver's container:
// /mnt/nvd/bind/<CONTAINER_ID>-<VOLUME_NAME>
func getContainerBindMountPath(volumeName, containerID string) string {
	mountPointName := fmt.Sprintf("%s-%s", volumeName, containerID)
	return filepath.Join(config.DriverMountPointsRoot, "bind", mountPointName)
}

// getNFSMountSource return NFS mount source to use in `mount` command
// Example: "10.3.199.243:/spool01/dataset/testvolume"
func getNFSMountSource(address, path string) string {
	return fmt.Sprintf("%s:/%s", address, strings.TrimPrefix(path, "/"))
}

// logError logs and returns the same error
func logError(l *logrus.Entry, err error) error {
	l.Error(err)
	return err
}
