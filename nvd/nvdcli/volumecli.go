package nvdcli

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/qeas/nexenta-docker-driver/nvd/nvdapi"
)


var (
	VolumeCmd =  cli.Command{
		Name:  "volume",
		Usage: "Volume related commands",
		Subcommands: []cli.Command{
			VolumeCreateCmd,
			VolumeDeleteCmd,
			VolumeListCmd,
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Enable verbose/debug logging: `[--verbose]`",
			},
			cli.StringFlag{
				Name:  "config, c",
				Usage: "Config file for daemon (default: /etc/nvd/nvd.json): `[--config /etc/nvd/nvd.json]`",
			},
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
			cli.StringFlag{
				Name:  "type",
				Usage: "Specify a volume type ",
			},
		},
		Action: cmdCreateVolume,
	}
	VolumeDeleteCmd = cli.Command{
		Name:  "delete",
		Usage: "delete an existing volume: `delete NAME`",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "range",
				Value: "",
				Usage: ": deletes a range of volume`",
			},
		},
		Action: cmdDeleteVolume,
	}
	VolumeListCmd = cli.Command{
		Name:  "list",
		Usage: "list existing volumes",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "range",
				Value: "",
				Usage: ": range of volume`",
			},
		},
		Action: cmdListVolumes,
	}

)

func getClient(c *cli.Context) (client *nvdapi.Client) {
	cfg := c.String("config")
	if cfg == "" {
		cfg = "/etc/nvd/nvd.json"
	}
	client, _ = nvdapi.ClientAlloc(cfg)
	return client
}

func cmdCreateVolume(c *cli.Context) cli.ActionFunc {
	fmt.Println("cmdCreate: ", c.String("name"));
	client := getClient(c)
	client.CreateVolume(c.String("name"))
	return nil
}

func cmdDeleteVolume(c *cli.Context) cli.ActionFunc {
	fmt.Println("cmdDeleteVolume: ", c.String("name"));
	client := getClient(c)
	client.DeleteVolume(c.String("name"))
	return nil
}

func cmdMountVolume(c *cli.Context) cli.ActionFunc {
	fmt.Println("cmdMountVolume: ", c.String("name"));
	client := getClient(c)
	client.MountVolume(c.String("name"))
	return nil
}

func cmdUnmountVolume(c *cli.Context) cli.ActionFunc {
	fmt.Println("cmdUnmountVolume: ", c.String("name"));
	client := getClient(c)
	client.UnmountVolume(c.String("name"))
	return nil
}

func cmdListVolumes(c *cli.Context) cli.ActionFunc {
	client := getClient(c)
	vols, err := client.ListVolumes()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("cmdListVolumes: ", vols);
	}
	return nil
}
