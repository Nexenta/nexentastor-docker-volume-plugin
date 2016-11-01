package nvdcli

import (
	"github.com/urfave/cli"
	nvdDaemon "github.com/Nexenta/nexenta-docker-driver/nvd/daemon"
	"github.com/sevlyar/go-daemon"
	log "github.com/Sirupsen/logrus"
	"syscall"
)

var (
	DaemonCmd = cli.Command{
		Name:  "daemon",
		Usage: "daemon related commands",
		Subcommands: []cli.Command{
			DaemonStartCmd,
			DaemonStopCmd,
		},
	}

	DaemonStartCmd = cli.Command{
		Name:  "start",
		Usage: "Start the Nexenta Docker Daemon: `start [options] NAME`",
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
		Action: cmdDaemonStart,
	}

	DaemonStopCmd = cli.Command{
		Name: "stop",
		Usage: "Stop the Nedge Docker Daemon: `stop",
		Action: cmdDaemonStop,
	}
)

func cmdDaemonStop(c *cli.Context) {
	cntxt := &daemon.Context{
		PidFileName: "/var/run/ndvol.pid",
		PidFilePerm: 0644,
		LogFileName: "/var/log/ndvol.log",
		LogFilePerm: 0640,
		Umask:       027,
	}
	d, err := cntxt.Search()
	if err != nil {
		log.Fatalln("Unable to send signal to the daemon:", err)
	}
	d.Signal(syscall.SIGTERM)
}

func cmdDaemonStart(c *cli.Context) {
	cntxt := &daemon.Context{
		PidFileName: "/var/run/ndvol.pid",
		PidFilePerm: 0644,
		LogFileName: "/var/log/ndvol.log",
		LogFilePerm: 0640,
		Umask:       027,
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	defer cntxt.Release()
	if d != nil {
		return
	}

	log.Info("- - - - - - - - - - - - - - -")
	log.Info("Daemon started")
	go DaemonStart(c)

	err = daemon.ServeSignals()
	if err != nil {
		log.Info("Error:", err)
	}
	log.Info("Daemon terminated")
}

func DaemonStart(c *cli.Context) {
	verbose := c.Bool("verbose")
	cfg := c.String("config")
	if cfg == "" {
		cfg = "/etc/nvd/nvd.json"
	}
	nvdDaemon.Start(cfg, verbose)
}
