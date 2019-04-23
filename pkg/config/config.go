package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Version - driver version, to set version set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/config.Version=1.0.0"
var Version string

// Commit - driver last commit, to set commit set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/config.Commit=..."
var Commit string

// DateTime - driver build datetime, to set commit set flags:
// go build -ldflags "-X github.com/Nexenta/nexenta-docker-driver/pkg/config.DateTime=..."
var DateTime string

// persistent driver's config
const (
	// Name - driver's executable name, must be the same as in `Makefile`
	Name = "nvd"

	// DriverMountPointsRoot - path inside the driver container to mount volumes
	// this path must be propagated to host via "propogatedmount" parameter in driver's "config.json"
	// TODO read this parameter from driver's "config.json" file "propogatedmount" parameter?
	DriverMountPointsRoot = "/mnt/nvd"

	// path to a log file inside the driver's container
	LogFile = "/var/log/nvd.log"
)

// supported mount filesystem types
const (
	// FsTypeNFS - to mount NS filesystem over NFS
	FsTypeNFS string = "nfs"
)

// a valid NS REST endpoint
const addressRegExp = "^https?://[^:]+:[0-9]{1,5}$"

// Config - driver config from file
type Config struct {
	Address             string `yaml:"restIp"`
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	DefaultDataset      string `yaml:"defaultDataset,omitempty"`
	DefaultDataIP       string `yaml:"defaultDataIp,omitempty"`
	Debug               bool   `yaml:"debug,omitempty"`
	DefaultMountOptions string `yaml:"defaultMountOptions,omitempty"`

	filePath    string
	lastMobTime time.Time
}

// New creates config instance
func New(configFilePath string) (*Config, error) {
	// read config file
	config := &Config{filePath: configFilePath}
	if _, err := config.Refresh(); err != nil {
		return nil, fmt.Errorf("Cannot refresh config from file '%s': %s", configFilePath, err)
	}

	return config, nil
}

// GetFilePath gets filepath of found config file
func (c *Config) GetFilePath() string {
	return c.filePath
}

// Refresh reads and validates config, returns `true` if config has been changed
func (c *Config) Refresh() (changed bool, err error) {
	if c.filePath == "" {
		return false, fmt.Errorf("Cannot read config file, filePath not specified")
	}

	fileInfo, err := os.Stat(c.filePath)
	if err != nil {
		return false, fmt.Errorf("Cannot get stats for '%s' config file: %s", c.filePath, err)
	}

	changed = c.lastMobTime != fileInfo.ModTime()

	if changed {
		c.lastMobTime = fileInfo.ModTime()

		content, err := ioutil.ReadFile(c.filePath)
		if err != nil {
			return changed, fmt.Errorf("Cannot read '%s' config file: %s", c.filePath, err)
		}

		if err := yaml.Unmarshal(content, c); err != nil {
			return changed, fmt.Errorf("Cannot parse yaml in '%s' config file: %s", c.filePath, err)
		}

		if err := c.Validate(); err != nil {
			return changed, err
		}
	}

	return changed, nil
}

// Validate validates current config
func (c *Config) Validate() error {
	var errors []string

	if c.Address == "" {
		errors = append(errors, fmt.Sprintf("parameter 'restIp' is missed"))
	} else {
		addresses := strings.Split(c.Address, ",")
		for _, address := range addresses {
			if !regexp.MustCompile(addressRegExp).MatchString(address) {
				errors = append(
					errors,
					fmt.Sprintf(
						"parameter 'restIp' has invalid address: '%s', should be 'schema://host:port'",
						address,
					),
				)
			}
		}
	}
	if c.Username == "" {
		errors = append(errors, fmt.Sprintf("parameter 'username' is missed"))
	}
	if c.Password == "" {
		errors = append(errors, fmt.Sprintf("parameter 'password' is missed"))
	}
	if c.DefaultDataset == "" {
		errors = append(errors, fmt.Sprintf("parameter 'defaultDataset' is missed"))
	}
	if c.DefaultDataIP == "" {
		errors = append(errors, fmt.Sprintf("parameter 'defaultDataIp' is missed"))
	}

	if len(errors) != 0 {
		return fmt.Errorf("Bad format, fix following issues: %s", strings.Join(errors, "; "))
	}

	return nil
}
