package nvdcli

import (
	"github.com/urfave/cli"
	"github.com/Nexenta/nexenta-docker-driver/nvd/nvdapi"
	log "github.com/Sirupsen/logrus"
)


var (
	VolumeCmd =  cli.Command{
		Name:  "volume",
		Usage: "Volume related commands",
		Subcommands: []cli.Command{
			VolumeCreateCmd,
			VolumeDeleteCmd,
			VolumeListCmd,
			VolumeMountCmd,
			VolumeUnmountCmd,
		},
	}

	VolumeCreateCmd = cli.Command{
		Name:  "create",
		Usage: "create a new volume: `create [options] NAME`",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "name",
				Usage: "volume name",
			},
			cli.StringFlag{
				Name:  "size",
				Usage: "size of volume in bytes ",
			},
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
		},
		Action: cmdCreateVolume,
	}
	VolumeDeleteCmd = cli.Command{
		Name:  "delete",
		Usage: "delete an existing volume: `delete NAME`",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
		},
		Action: cmdDeleteVolume,
	}
	VolumeListCmd = cli.Command{
		Name:  "list",
		Usage: "list existing volumes",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
		},
		Action: cmdListVolumes,
	}
	VolumeMountCmd = cli.Command{
		Name: "mount",
		Usage: "mount an existing volume: `mount NAME`",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
		},
		Action: cmdMountVolume,
	}
	VolumeUnmountCmd = cli.Command{
		Name: "unmount",
		Usage: "unmount an existing volume: `unmount NAME`",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
		},
		Action: cmdUnmountVolume,
	}
)

func getClient(c *cli.Context) (client *nvdapi.Client) {
	cfg := c.String("config")
	if cfg == "" {
		cfg = "/etc/nvd/nvd.json"
	}
	if c.Bool("v") == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	
	client, _ = nvdapi.ClientAlloc(cfg)
	return client
}

func cmdCreateVolume(c *cli.Context) cli.ActionFunc {
	name := c.Args().First()
	if name == "" {
		log.Error("Provide volume name as first argument")
		return nil
	}
	log.Debug("cmdCreate: ", name);
	client := getClient(c)
	client.CreateVolume(name)
	return nil
}

func cmdDeleteVolume(c *cli.Context) cli.ActionFunc {
	log.Debug("cmdDeleteVolume: ", c.String("name"));
	client := getClient(c)
	client.DeleteVolume(c.String("name"))
	return nil
}

func cmdMountVolume(c *cli.Context) cli.ActionFunc {
	log.Debug("cmdMountVolume: ", c.String("name"));
	client := getClient(c)
	client.MountVolume(c.String("name"))
	return nil
}

func cmdUnmountVolume(c *cli.Context) cli.ActionFunc {
	log.Debug("cmdUnmountVolume: ", c.String("name"));
	client := getClient(c)
	client.UnmountVolume(c.String("name"))
	return nil
}

func cmdListVolumes(c *cli.Context) cli.ActionFunc {
	client := getClient(c)
	vols, err := client.ListVolumes()
	if err != nil {
		log.Debug(err)
	} else {
		log.Debug("cmdListVolumes: ", vols);
	}
	return nil
}
