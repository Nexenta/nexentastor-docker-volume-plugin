package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	nestedLogrusFormatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"

	"github.com/Nexenta/nexenta-docker-driver/pkg/config"
	"github.com/Nexenta/nexenta-docker-driver/pkg/driver"
)

const (
	defaultConfigFile    = "/etc/nvd/nvd.yaml"
	defaultSocketAddress = "/run/docker/plugins/nvd.sock"
)

func main() {
	var (
		configFile = flag.String("config", defaultConfigFile, "driver config file")
		version    = flag.Bool("version", false, "print driver version")
	)

	flag.Parse()

	if *version {
		fmt.Printf("%s@%s-%s (%s)\n", driver.Name, driver.Version, driver.Commit, driver.DateTime)
		os.Exit(0)
	}

	// init logger
	l := initLogger()

	l.Infof("%s@%s-%s (%s) started...", driver.Name, driver.Version, driver.Commit, driver.DateTime)
	l.Info("run driver with CLI options:")
	l.Infof("- Config file: '%s'", *configFile)

	// initial read and validate config file
	cfg, err := config.New(*configFile)
	if err != nil {
		l.Fatalf("Cannot use config file: %s", err)
	}

	// logger level
	if cfg.Debug {
		l.Logger.SetLevel(logrus.DebugLevel)
	} else {
		l.Logger.SetLevel(logrus.InfoLevel)
	}

	l.Info("config file options:")
	l.Infof("- NexentaStor address(es): %s", cfg.Address)
	l.Infof("- NexentaStor username: %s", cfg.Username)
	l.Infof("- Default dataset: %s", cfg.DefaultDataset)
	l.Infof("- Default data IP: %s", cfg.DefaultDataIP)
	l.Infof("- Debug: %t", cfg.Debug)

	// create driver
	d, err := driver.New(driver.Args{
		Config: cfg,
		Log:    l,
	})
	if err != nil {
		l.Fatalf("Failed to create volume driver: %s", err)
	}

	l.Infof("run server on '%s'...", defaultSocketAddress)
	handler := volume.NewHandler(d)
	err = handler.ServeUnix(defaultSocketAddress, 0)
	if err != nil {
		l.Fatalf("Failed to start server: %s", err)
	} else {
		l.Info("Driver process terminated.")
	}
}

func initLogger() *logrus.Entry {
	l := logrus.New().WithFields(logrus.Fields{
		"driver": fmt.Sprintf("%s@%s", driver.Name, driver.Version),
		"cmp":    "Main",
	})

	// set logger formatter
	l.Logger.SetFormatter(&nestedLogrusFormatter.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"driver", "cmp", "ns", "func", "req", "reqID", "job"},
	})

	logFile := "/var/log/nvd.log"
	logFileWriter, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		l.Fatal("log")
		l.Logger.SetOutput(os.Stdout)
		return l
	}

	mw := io.MultiWriter(os.Stdout, logFileWriter)
	l.Logger.SetOutput(mw)

	return l
}
