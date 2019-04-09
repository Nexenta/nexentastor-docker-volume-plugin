package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/util/mount"

	"github.com/Nexenta/nexenta-docker-driver/pkg/arrays"
	"github.com/Nexenta/nexenta-docker-driver/pkg/config"
	"github.com/Nexenta/nexentastor-csi-driver/pkg/ns" //TODO move to a dedicated library
)

// Version - driver version, to set version set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.Version=1.0.0"
var Version string

// Commit - driver last commit, to set commit set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.Commit=..."
var Commit string

// DateTime - driver build datetime, to set commit set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/driver.DateTime=..."
var DateTime string

const (
	// Name - driver's executable name
	Name = "nvd"

	//TODO this path should be read from driver's "config.json" file "propogatedmount" parameter
	driverMountPointsRoot = "/mnt/nvd"
)

// Driver for NS
type Driver struct {
	log        *logrus.Entry
	config     *config.Config
	nsResolver *ns.Resolver
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
	}, nil
}

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

func (d *Driver) resolveNS(datasetPath string) (ns.ProviderInterface, error) {
	nsProvider, err := d.nsResolver.Resolve(datasetPath)
	if err != nil {
		code := "Internal Error"
		if ns.IsNotExistNefError(err) {
			code = "Not Found"
		}
		return nil, fmt.Errorf(
			"%s: Cannot resolve '%s' on any NexentaStor(s): %s",
			code,
			datasetPath,
			err,
		)
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

// Create Docker volume which is backed on NS filesystem
func (d *Driver) Create(req *volume.CreateRequest) error {
	l := d.log.WithField("func", "Create()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	err := d.refreshConfig()
	if err != nil {
		return fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	nsProvider, err := d.resolveNS(datasetPath)
	if err != nil {
		l.Error(err)
		return err
	}

	l.Infof("resolved NS: %s, %s", nsProvider, datasetPath)

	// use full path as volume ID
	volumePath := filepath.Join(datasetPath, volumeName)

	err = nsProvider.CreateFilesystem(ns.CreateFilesystemParams{
		Path: volumePath,
	})
	if err != nil {
		if ns.IsAlreadyExistNefError(err) {
			l.Infof("volume '%s' already exists and can be used", volumePath)
			return nil
		}

		return fmt.Errorf(
			"InternalError: Cannot create volume '%s': %s",
			volumePath,
			err,
		)
	}

	l.Infof("volume '%s' has been created", volumePath)

	return nil
}

// Remove Docker volume and its NS filesystem
func (d *Driver) Remove(req *volume.RemoveRequest) error {
	l := d.log.WithField("func", "Remove()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	err := d.refreshConfig()
	if err != nil {
		return fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	// use full path as volume ID
	volumePath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(volumePath)
	if err != nil {
		l.Error(err)
		return err
	}

	l.Infof("resolved NS: %s, %s", nsProvider, volumePath)

	// if here, than volumePath exists on some NS
	err = nsProvider.DestroyFilesystemWithClones(volumePath, false)
	if err != nil && !ns.IsNotExistNefError(err) {
		return fmt.Errorf("Cannot delete '%s' volume: %s", volumePath, err)
	}

	l.Infof("volume '%s' has been deleted", volumePath)
	return nil
}

// List Docker volumes managed by NS
func (d *Driver) List() (*volume.ListResponse, error) {
	l := d.log.WithField("func", "List()")
	l.Infof("request")

	err := d.refreshConfig()
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	nsProvider, err := d.resolveNS(datasetPath)
	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("resolved NS: %s, %s", nsProvider, datasetPath)

	filesystems, err := nsProvider.GetFilesystems(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot get filesystems: %s", err)
	}

	volumes := make([]*volume.Volume, len(filesystems))
	for i, item := range filesystems {
		name := strings.TrimPrefix(item.Path, datasetPath+"/")
		volumes[i] = &volume.Volume{
			Name:       name,
			Mountpoint: filepath.Join(driverMountPointsRoot, name),
		}
	}

	l.Infof("found %d entries(s)", len(volumes))

	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

// Get Docker volume by name
func (d *Driver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	l := d.log.WithField("func", "Get()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return nil, fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	err := d.refreshConfig()
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	// use full path as volume ID
	volumePath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(volumePath)
	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("resolved NS: %s, %s", nsProvider, datasetPath)

	filesystem, err := nsProvider.GetFilesystem(volumePath)
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot get filesystem '%s': %s", volumePath, err)
	}

	l.Infof("filesystem '%s' was found", filesystem.String())

	name := strings.TrimPrefix(filesystem.Path, datasetPath+"/") //TODO add name to API
	mountPoint := filepath.Join(driverMountPointsRoot, name)

	return &volume.GetResponse{
		Volume: &volume.Volume{
			Name:       name,
			Mountpoint: mountPoint,
		},
	}, nil
}

// Path returns Docker volume mount path
func (d *Driver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	l := d.log.WithField("func", "Path()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return nil, fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	mountPoint := filepath.Join(driverMountPointsRoot, volumeName)

	return &volume.PathResponse{
		Mountpoint: mountPoint,
	}, nil
}

// Mount NS filesystem to local path for Docker volume
func (d *Driver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	l := d.log.WithField("func", "Mount()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return nil, fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	// read and validate config
	err := d.refreshConfig()
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	// path on host file system
	targetPath := filepath.Join(driverMountPointsRoot, volumeName)

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	// use full path as volume ID
	volumePath := filepath.Join(datasetPath, volumeName)

	nsProvider, err := d.resolveNS(volumePath)
	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("resolved NS: %s, %s", nsProvider, volumePath)

	// get NexentaStor filesystem information
	filesystem, err := nsProvider.GetFilesystem(volumePath)
	if err != nil {
		return nil, fmt.Errorf("FailedPrecondition: Cannot get filesystem '%s': %s", volumePath, err)
	}

	//TODO get mount options from runtime params, set default if not specified
	mountOptions := []string{}
	for _, option := range strings.Split(d.config.DefaultMountOptions, ",") {
		if option != "" {
			mountOptions = append(mountOptions, option)
		}
	}

	//TODO get dataIp from runtime params, set default if not specified
	dataIP := d.config.DefaultDataIP // `defaultDataIP` in driver config file

	//TODO get fsType from runtime params, set default if not specified
	var fsType string
	//if d.config.DefaultMountFsType != "" {
	//	fsType = d.config.DefaultMountFsType
	//} else {
	fsType = config.FsTypeNFS
	//}

	// share and mount filesystem with selected type
	if fsType == config.FsTypeNFS {
		err = d.mountNFS(nsProvider, filesystem, dataIP, mountOptions, targetPath)
		//} else if fsType == config.FsTypeCIFS {
		//TODO CIFS mount
	} else {
		err = fmt.Errorf("FailedPrecondition: Unsupported mount filesystem type: '%s'", fsType)
	}
	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("volume '%s' has been mounted to '%s'", volumePath, targetPath)
	return &volume.MountResponse{
		Mountpoint: targetPath,
	}, nil
}

func (d *Driver) mountNFS(
	nsProvider ns.ProviderInterface,
	filesystem ns.Filesystem,
	dataIP string,
	mountOptions []string,
	targetPath string,
) error {
	// create NFS share if not exists
	if !filesystem.SharedOverNfs {
		err := nsProvider.CreateNfsShare(ns.CreateNfsShareParams{
			Filesystem: filesystem.Path,
		})
		if err != nil {
			return fmt.Errorf("InternalError: Cannot share filesystem '%s' over NFS: %s", filesystem.Path, err)
		}

		// TODO select read-only or read-write mount options set
		var aclRuleSet ns.ACLRuleSet
		aclRuleSet = ns.ACLReadWrite
		// if req.GetReadonly() {
		// 	aclRuleSet = ns.ACLReadOnly
		// } else {
		// 	aclRuleSet = ns.ACLReadWrite
		// }

		// apply NS filesystem ACL (gets applied only for new volumes, not for already shared volumes)
		err = nsProvider.SetFilesystemACL(filesystem.Path, aclRuleSet)
		if err != nil {
			return fmt.Errorf("InternalError: Cannot set filesystem ACL for '%s': %s", filesystem.Path, err)
		}
	}

	// NFS style mount source
	mountSource := fmt.Sprintf("%s:%s", dataIP, filesystem.MountPoint)

	// NFS v3 is used by default if no version specified by user
	mountOptions = arrays.AppendIfRegexpNotExistString(mountOptions, regexp.MustCompile("^vers=.*$"), "vers=3")

	return d.doMount(mountSource, targetPath, config.FsTypeNFS, mountOptions)
}

func (d *Driver) doMount(mountSource, targetPath, fsType string, mountOptions []string) error {
	l := d.log.WithField("func", "doMount()")

	mounter := mount.New("")

	// check if mountpoint exists, create if there is no such directory
	notMountPoint, err := mounter.IsLikelyNotMountPoint(targetPath)
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

	l.Infof(
		"mount params: type: '%s', mountSource: '%s', targetPath: '%s', mountOptions(%v): %+v",
		fsType,
		targetPath,
		mountSource,
		len(mountOptions),
		mountOptions,
	)

	err = mounter.Mount(mountSource, targetPath, fsType, mountOptions)
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

// Unmount - un-mount volume
func (d *Driver) Unmount(req *volume.UnmountRequest) error {
	l := d.log.WithField("func", "Unmount()")
	l.Infof("request: '%+v'", req)

	volumeName := req.Name
	if len(volumeName) == 0 {
		return fmt.Errorf("InvalidArgument: req.Name must be provided")
	}

	// read and validate config
	err := d.refreshConfig()
	if err != nil {
		return fmt.Errorf("FailedPrecondition: Cannot use config file: %s", err)
	}

	// path on host file system
	targetPath := filepath.Join(driverMountPointsRoot, volumeName)

	//TODO get dataset path from runtime params, set default if not specified
	datasetPath := d.config.DefaultDataset

	// use full path as volume ID
	volumePath := filepath.Join(datasetPath, volumeName)

	mounter := mount.New("")

	if err := mounter.Unmount(targetPath); err != nil {
		return fmt.Errorf("InternalError: Failed to unmount target path '%s': %s", targetPath, err)
	}

	notMountPoint, err := mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			l.Warnf("mount point '%s' already doesn't exist: '%s', return OK", targetPath, err)
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

	l.Infof("volume '%s' has been unpublished from '%s'", volumePath, targetPath)
	return nil
}
