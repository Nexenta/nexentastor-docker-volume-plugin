package main

import (
	"github.com/qeas/nexenta-docker-driver/nvd/nvdapi"
	"github.com/qeas/nexenta-docker-driver/nvd/nvdcli"
	"os"
)

const (
	VERSION = "0.0.1"
)

var (
	client *nvdapi.Client
)

func main() {
	ncli := nvdcli.NewCli(VERSION)
	ncli.Run(os.Args)
}

