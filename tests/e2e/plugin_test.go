package plugin_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	pluginConfig "github.com/Nexenta/nexentastor-docker-volume-plugin/pkg/config"
	"github.com/Nexenta/nexentastor-docker-volume-plugin/tests/utils/docker"
	"github.com/Nexenta/nexentastor-docker-volume-plugin/tests/utils/remote"
)

var (
	volumeName = "e2eTestsVolume"
)

type config struct {
	sshConnectionString string
	pluginName          string
	configPath          string
	mountSourceBase     string
}

var l *logrus.Entry
var c *config
var pc *pluginConfig.Config

func TestMain(m *testing.M) {
	var (
		argSSH    = flag.String("ssh", "", "ssh connections string to Docker setup [user@host]")
		argPlugin = flag.String("plugin", "", "Full Docker plugin name to test [repository/image:tag]")
		argConfig = flag.String("config", "", "path to config file")
	)

	flag.Parse()

	if *argSSH == "" {
		fmt.Println("Parameter '--ssh' is missed")
		os.Exit(1)
	} else if *argConfig == "" {
		fmt.Println("Parameter '--plugin' is missed")
		os.Exit(1)
	} else if *argConfig == "" {
		fmt.Println("Parameter '--config' is missed")
		os.Exit(1)
	}

	c = &config{
		sshConnectionString: *argSSH,
		pluginName:          *argPlugin,
		configPath:          *argConfig,
	}

	// init logger
	l = logrus.New().WithField("title", "tests")

	noColors := false
	if v := os.Getenv("NOCOLORS"); v != "" && v != "false" {
		noColors = true
	}

	// logger formatter
	l.Logger.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"title", "address", "cmp", "func"},
		NoColors:    noColors,
	})

	filePluginConfig, err := pluginConfig.New(c.configPath)
	if err != nil {
		fmt.Printf("Cannot use config file '%s': %s\n", c.configPath, err)
		os.Exit(1)
	}
	pc = filePluginConfig // make it global

	l.Info("run...")

	os.Exit(m.Run())
}

func TestPlugin_deploy(t *testing.T) {
	rc, err := remote.NewClient(c.sshConnectionString, l)
	if err != nil {
		t.Errorf("Cannot create connection: %s", err)
		return
	}

	dockerPlugin, err := docker.NewPluginDeployment(docker.DeploymentArgs{
		RemoteClient: rc,
		PluginName:   c.pluginName,
		ConfigPath:   c.configPath,
		Log:          l,
	})
	if err != nil {
		t.Errorf("Cannot create Docker plugin deployment: %s", err)
		return
	}
	defer dockerPlugin.CleanUp()

	t.Run("install plugin", func(t *testing.T) {
		if err := dockerPlugin.Install(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run(fmt.Sprintf("create volume: %s", volumeName), func(t *testing.T) {
		dockerPlugin.RemoveVolume(volumeName)
		if err := dockerPlugin.CreateVolume(volumeName); err != nil {
			t.Fatal(err)
		}
	})

	// Test scenario:
	//  - run container with the volume mounted and writes file to the volume (memorize MD5 of the file)
	//  - run another container with the volume mounted and memorize MD5 of the file
	//  - mounts NS filesystem on the host and memorize MD5 of the file
	//  - compare all 3 MD5 hash sums to validate them (should be equal)
	t.Run("should write data to NS and provide it for all containers", func(t *testing.T) {
		fileName := "big"
		containerFilePath := filepath.Join("/mnt", volumeName, fileName)

		// run container, create file and calculate its md5 hash
		md5Command := fmt.Sprintf("md5sum %s", containerFilePath)
		ddAndMd5Command := fmt.Sprintf(
			"dd if=/dev/urandom of=%s bs=1M count=10 >& /dev/null && %s",
			containerFilePath,
			md5Command,
		)
		fileMd5A, err := dockerPlugin.RunVolumeContainerCommand(volumeName, ddAndMd5Command)
		if err != nil {
			t.Fatal(err)
		}
		fileMd5A = fileMd5A[0:32]
		l.Infof("Md5 hash in first container, for file '%s': %s", containerFilePath, fileMd5A)

		// run another container, get md5 hash again
		fileMd5B, err := dockerPlugin.RunVolumeContainerCommand(volumeName, md5Command)
		if err != nil {
			t.Fatal(err)
		}
		fileMd5B = fileMd5B[0:32]
		l.Infof("Md5 hash in second container, for file '%s': %s", containerFilePath, fileMd5B)

		// compare md5 hashes inside containers
		if fileMd5A != fileMd5B {
			t.Fatalf(
				"File '%s' should have same md5 hash sums in different containers, but got '%s' and '%s'",
				containerFilePath,
				fileMd5A,
				fileMd5B,
			)
		}

		// create mount point and mount same volume without using docker
		mountSource := fmt.Sprintf("%s:%s/%s", pc.DefaultDataIP, pc.DefaultDataset, volumeName)
		mountPoint := filepath.Join("/tmp", volumeName)
		mountPointFilePath := filepath.Join(mountPoint, fileName)
		mountCleanUp := func() {
			rc.Exec(fmt.Sprintf("umount %s | true", mountPoint))
			rc.Exec(fmt.Sprintf("rm -rf %s | true", mountPoint))
		}
		out, err := rc.Exec(fmt.Sprintf("showmount -e %s", pc.DefaultDataIP))
		if err != nil {
			t.Fatal(err)
		}
		l.Infof("Verify mounts:\n%s", out)
		_, err = rc.Exec(fmt.Sprintf("mkdir -p %s", mountPoint))
		if err != nil {
			t.Fatal(err)
		}
		_, err = rc.Exec(fmt.Sprintf("mount -t nfs -o vers=3 %s %s", mountSource, mountPoint))
		if err != nil {
			mountCleanUp()
			t.Fatal(err)
		}
		fileMd5C, err := rc.Exec(fmt.Sprintf("md5sum %s", mountPointFilePath))
		if err != nil {
			mountCleanUp()
			t.Fatal(err)
		}
		fileMd5C = fileMd5C[0:32]
		l.Infof("Md5 hash on host, for file '%s': %s", mountPointFilePath, fileMd5C)
		// compare md5 hashes inside and outside of container
		if fileMd5A != fileMd5C {
			mountCleanUp()
			t.Fatalf(
				"File inside container (%s:%s) has different md5 hash that file mounted outside of container "+
					"'%s:%s'",
				containerFilePath,
				fileMd5A,
				mountPointFilePath,
				fileMd5C,
			)
		}
		mountCleanUp()

		l.Info("OK: All files have same md5 hash")
	})

	// Test scenario:
	//  - many containers use same the volume at the same time
	//  - volume gets unmounted when no container uses it
	t.Run("should be able to run many containers with the same volume mounted", func(t *testing.T) {
		sleepSeconds := 15
		sleepCommand := fmt.Sprintf("sleep %d", sleepSeconds)

		// containers to run in parallel
		containerCount := 5

		l.Infof("Run %d containers...", containerCount)
		var wg sync.WaitGroup
		result := struct {
			*sync.Mutex
			errors []string
		}{}
		for i := 1; i <= containerCount; i++ {
			wg.Add(1)
			go func(i int) {
				l.Infof("Start container #%d...", i)
				_, err := dockerPlugin.RunVolumeContainerCommand(volumeName, sleepCommand)
				if err != nil {
					result.Lock()
					result.errors = append(result.errors, err.Error())
					result.Unlock()
					l.Errorf("Container #%d error: %s", i, err)
				}
				l.Infof("Done container #%d.", i)
				wg.Done()
			}(i)
		}
		l.Infof("Waiting for containers to exit in %ds...", sleepSeconds)

		// print `docker ps`
		go func() {
			d := time.Duration(sleepSeconds/2) * time.Second
			time.Sleep(d)
			out, _ := rc.Exec("docker ps | grep ubuntu")
			l.Infof("Running containers at %s:\n%s", d, out)
		}()

		wg.Wait()
		l.Info("All containers are exited.")

		if len(result.errors) > 0 {
			t.Fatalf("failed to run %d containers, errors:\n%s", containerCount, strings.Join(result.errors, "\n"))
		}

		// check if mount point is still mounted
		mountSource := fmt.Sprintf("%s:%s/%s", pc.DefaultDataIP, pc.DefaultDataset, volumeName)
		out, err := rc.Exec(fmt.Sprintf("cat /proc/mounts | grep %s", mountSource))
		if err == nil {
			t.Fatalf("Mount point still found for source '%s': %s", mountSource, out)
		}

		l.Info("OK: All files have same md5 hash")
	})

	t.Run(fmt.Sprintf("remove volume: %s", volumeName), func(t *testing.T) {
		if err := dockerPlugin.RemoveVolume(volumeName); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("uninstall docker plugin", func(t *testing.T) {
		if err := dockerPlugin.Uninstall(); err != nil {
			t.Fatal(err)
		}
	})
}
