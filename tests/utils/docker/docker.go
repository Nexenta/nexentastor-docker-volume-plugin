package docker

import (
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Nexenta/nexenta-docker-driver/tests/utils/remote"
)

const (
	// plugin config file on remote Docker setup
	pluginConfigPath = "/etc/nvd/nvd.yaml"
)

// PluginDeployment - Docker plugin deployment
type PluginDeployment struct {
	// RemoteClient - ssh client to connect through
	RemoteClient *remote.Client

	// PluginName - Docker plugin name to deploy
	PluginName string

	// ConfigPath - path to yaml config file for docker plugin
	ConfigPath string

	log *logrus.Entry
}

func (d *PluginDeployment) createFormattedError(message string) func(error) error {
	return func(err error) error {
		fullErr := fmt.Errorf(
			"Docker plugin deployment failed (%s on %v): %s: %s",
			d.PluginName,
			d.RemoteClient,
			message,
			err,
		)
		d.log.Error(fullErr)
		return fullErr
	}
}

// Install - runs `docker plugin install ...` and `docker plugin enable ...`
func (d *PluginDeployment) Install() error {
	l := d.log.WithField("func", "Install()")
	fail := d.createFormattedError("Install()")

	l.Info(d.PluginName)

	// save a copy of original config file (ignore if does not exist)
	d.saveOriginalConfig()

	// copy local config to remote plugin config file
	if err := d.RemoteClient.CopyFiles(d.ConfigPath, pluginConfigPath); err != nil {
		return fail(err)
	}

	// install plugin
	installCommand := fmt.Sprintf("docker plugin install --grant-all-permissions --disable %s", d.PluginName)
	if _, err := d.RemoteClient.Exec(installCommand); err != nil {
		return fail(err)
	}

	// enable plugin
	if _, err := d.RemoteClient.Exec(fmt.Sprintf("docker plugin enable %s", d.PluginName)); err != nil {
		return fail(err)
	}

	l.Infof("plugin %s has been successfully installed to %s", d.PluginName, d.RemoteClient)
	return nil
}

// CreateVolume creates Docker volume
func (d *PluginDeployment) CreateVolume(volume string) error {
	l := d.log.WithField("func", "CreateVolume()")
	fail := d.createFormattedError("CreateVolume()")

	l.Info(d.PluginName)

	// create volume
	createVolumeCommand := fmt.Sprintf("docker volume create -d %s --name=%s", d.PluginName, volume)
	if _, err := d.RemoteClient.Exec(createVolumeCommand); err != nil {
		return fail(err)
	}

	l.Infof("volume %s has been successfully created using %s", volume, d.PluginName)
	return nil
}

// RemoveVolume removes volume
func (d *PluginDeployment) RemoveVolume(volume string) error {
	l := d.log.WithField("func", "RemoveVolume()")
	fail := d.createFormattedError("RemoveVolume()")

	l.Info(d.PluginName)

	// remove volume
	removeVolumeCommand := fmt.Sprintf("docker volume remove %s", volume)
	if _, err := d.RemoteClient.Exec(removeVolumeCommand); err != nil {
		return fail(err)
	}

	l.Infof("volume %s has been successfully removed", volume)
	return nil
}

//RunVolumeContainerCommand runs Ubuntu Docker container with specified volume and executes command inside of it
func (d *PluginDeployment) RunVolumeContainerCommand(volume, command string) (string, error) {
	l := d.log.WithField("func", "RunVolumeContainerCommand()")
	fail := d.createFormattedError("RunVolumeContainerCommand()")

	l.Info(d.PluginName)

	// run container
	runContainerCommand := fmt.Sprintf(
		"docker run -v %s:/mnt/%s -t --rm ubuntu /bin/bash -c \"%s\"",
		volume,
		volume,
		command,
	)

	out, err := d.RemoteClient.Exec(runContainerCommand)
	if err != nil {
		return out, fail(err)
	}
	return out, nil
}

// Uninstall - runs `docker plugin disable -f ...` and `docker plugin rm -f ...`
// TODO use upgrade?
func (d *PluginDeployment) Uninstall() error {
	l := d.log.WithField("func", "Uninstall()")
	fail := d.createFormattedError("Uninstall()")

	l.Info(d.PluginName)

	// disable plugin
	if _, err := d.RemoteClient.Exec(fmt.Sprintf("docker plugin disable -f %s", d.PluginName)); err != nil {
		return fail(err)
	}

	// remove plugin
	if _, err := d.RemoteClient.Exec(fmt.Sprintf("docker plugin rm -f %s", d.PluginName)); err != nil {
		return fail(err)
	}

	// restore a copy of original config file (ignore if does not exist)
	d.restoreOriginalConfig()

	l.Infof("plugin %s has been successfully uninstalled from %s", d.PluginName, d.RemoteClient)
	return nil
}

// CleanUp - silently removes plugin and restores original config file
func (d *PluginDeployment) CleanUp() {
	l := d.log.WithField("func", "CleanUp()")

	l.Info(d.PluginName)

	// disable plugin
	d.RemoteClient.Exec(fmt.Sprintf("docker plugin disable -f %s | true", d.PluginName))

	// remove plugin
	d.RemoteClient.Exec(fmt.Sprintf("docker plugin rm -f %s | true", d.PluginName))

	// restore a copy of original config file (ignore if does not exist)
	d.restoreOriginalConfig()

	l.Info("done.")
}

// save a copy of original config file (ignore if does not exist)
func (d *PluginDeployment) saveOriginalConfig() {
	_, err := d.RemoteClient.Exec(fmt.Sprintf("test -f %s.original", pluginConfigPath))

	// if backup of original config doesn't exist, create it
	if err != nil {
		d.RemoteClient.Exec(fmt.Sprintf("cp %s %s.original", pluginConfigPath, pluginConfigPath))
	}
}

// restore a copy of original config file (ignore if it does not exist)
func (d *PluginDeployment) restoreOriginalConfig() {
	d.RemoteClient.Exec(fmt.Sprintf("cp %s.original %s", pluginConfigPath, pluginConfigPath))
	d.RemoteClient.Exec(fmt.Sprintf("rm %s.original", pluginConfigPath))
}

// DeploymentArgs - arguments for plugin deployment
type DeploymentArgs struct {
	RemoteClient *remote.Client
	PluginName   string
	ConfigPath   string
	Log          *logrus.Entry
}

// NewPluginDeployment - create new Docker plugin deployment
func NewPluginDeployment(args DeploymentArgs) (*PluginDeployment, error) {
	if args.RemoteClient == nil {
		return nil, fmt.Errorf("args.RemoteClient is required")
	} else if args.PluginName == "" {
		return nil, fmt.Errorf("args.PluginName is required")
	} else if args.ConfigPath == "" {
		return nil, fmt.Errorf("args.ConfigPath is required")
	} else if args.Log == nil {
		return nil, fmt.Errorf("args.Log is required")
	}

	l := args.Log.WithFields(logrus.Fields{
		"address": args.RemoteClient.ConnectionString,
		"cmp":     "docker",
	})

	pluginConfigDir := filepath.Dir(pluginConfigPath)
	if _, err := args.RemoteClient.Exec(fmt.Sprintf("mkdir -p %s", pluginConfigDir)); err != nil {
		return nil, fmt.Errorf(
			"NewDeployment(): cannot create '%s' directory on %+v",
			pluginConfigDir,
			args.RemoteClient,
		)
	}

	return &PluginDeployment{
		RemoteClient: args.RemoteClient,
		PluginName:   args.PluginName,
		ConfigPath:   args.ConfigPath,
		log:          l,
	}, nil
}
