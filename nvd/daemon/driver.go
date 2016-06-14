package daemon

import (
	log "github.com/Sirupsen/logrus"
	"sync"
	"github.com/docker/go-plugins-helpers/volume"
	"path/filepath"
	"github.com/qeas/nexenta-docker-driver/nvd/nvdapi"
)

type NexentaDriver struct {
	DefaultVolSz   int64
	MountPoint     string
	InitiatorIFace string
	Client         *nvdapi.Client
	Mutex          *sync.Mutex
}

func DriverAlloc(cfgFile string) NexentaDriver {

	client, _ := nvdapi.ClientAlloc(cfgFile)

	initiator := "NFS"

	d := NexentaDriver{
		DefaultVolSz:	1024,
		Client:         client,
		Mutex:          &sync.Mutex{},
		MountPoint:     client.MountPoint,
		InitiatorIFace: initiator,
	}

	return d
}

func (d NexentaDriver) Create(r volume.Request) volume.Response {

	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	err := d.Client.CreateVolume(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d NexentaDriver) Get(r volume.Request) volume.Response {
	path := d.Client.MountPoint + "/" + r.Name
	name, err := d.Client.GetVolume(r.Name)
	if err != nil {
		log.Info("Failed to retrieve volume named ", r.Name, "during Get operation: ", err)
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Volume: &volume.Volume{Name: name, Mountpoint: path}}
}

func (d NexentaDriver) List(r volume.Request) volume.Response {
	vlist, err := d.Client.ListVolumes()
	if err != nil {
		log.Error(err)
		return volume.Response{Err: err.Error()}
	}
	var vols []*volume.Volume
	for _, name := range vlist {
		if name != "" {
			vols = append(vols, &volume.Volume{Name: name, Mountpoint: filepath.Join(d.Client.MountPoint, name)})
		}
	}
	return volume.Response{Volumes: vols}
}

func (d NexentaDriver) Mount(r volume.Request) volume.Response {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	mnt := d.Client.MountPoint + "/" + r.Name
	d.Client.MountVolume(r.Name)
	return volume.Response{Mountpoint: mnt}
}

func (d NexentaDriver) Path(r volume.Request) volume.Response {
	log.Debug("Retrieve path info for volume: ", r.Name)
	path := filepath.Join(d.Client.MountPoint, r.Name)
	log.Debug("Path reported as: ", path)
	return volume.Response{Mountpoint: path}
}

func (d NexentaDriver) Remove(r volume.Request) volume.Response {
	d.Client.DeleteVolume(r.Name)
	return volume.Response{}
}

func (d NexentaDriver) Unmount(r volume.Request) volume.Response {
	d.Client.UnmountVolume(r.Name)
	return volume.Response{}
}

