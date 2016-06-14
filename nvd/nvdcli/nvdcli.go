package nvdcli

import (
	"fmt"
	"github.com/codegangsta/cli"
)


func NvdCmdNotFound(c *cli.Context, command string) {
	fmt.Println(command, " not found ");

}

func NvdInitialize(c *cli.Context) error {

	cfgFile := c.GlobalString("config")
	fmt.Println(cfgFile)
	if cfgFile != "" {
		fmt.Println("Found config: ", cfgFile);
	}
	return nil
}

func NewCli(version string) *cli.App {
	app := cli.NewApp()
	app.Name = "nvd"
	app.Version = version
	app.Author = "nexentaedge@nexenta.com"
	app.Usage = "CLI for Nexenta clusters"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "loglevel",
			Value:  "info",
			Usage:  "Specifies the logging level (debug|warning|error)",
			EnvVar: "LogLevel",
		},
	}
	app.CommandNotFound = NvdCmdNotFound
	app.Before = NvdInitialize
	app.Commands = []cli.Command{
		DaemonCmd,
		VolumeCmd,
	}
	return app
}
