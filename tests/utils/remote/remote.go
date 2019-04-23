package remote

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// Client - wrapper to run bash commands over ssh
type Client struct {
	// ConnectionString - user@host for ssh command
	ConnectionString string

	log *logrus.Entry
}

func (c *Client) String() string {
	return c.ConnectionString
}

// Exec - run command over ssh
func (c *Client) Exec(cmd string) (string, error) {
	l := c.log.WithField("func", "Exec()")
	l.Info(cmd)

	out, err := exec.Command("ssh", c.ConnectionString, cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command 'ssh %s %s' error: %s; out: %s", c.ConnectionString, cmd, err, out)
	}
	return fmt.Sprintf("%s", out), nil
}

// CopyFiles - copy local files to remote server
func (c *Client) CopyFiles(from, to string) error {
	l := c.log.WithField("func", "CopyFiles()")

	toAddress := fmt.Sprintf("%s:%s", c.ConnectionString, to)

	l.Infof("scp %s %s\n", from, toAddress)

	if out, err := exec.Command("scp", from, toAddress).CombinedOutput(); err != nil {
		return fmt.Errorf("Command 'scp %s %s' error: %s; out: %s", from, toAddress, err, out)
	}

	return nil
}

// NewClient - create new SSH remote client
func NewClient(connectionString string, log *logrus.Entry) (*Client, error) {
	l := log.WithFields(logrus.Fields{
		"address": connectionString,
		"cmp":     "remote",
	})

	client := &Client{
		ConnectionString: connectionString,
		log:              l,
	}

	_, err := client.Exec("date")
	if err != nil {
		return nil, fmt.Errorf("Failed to validate %s connection: %s", connectionString, err)
	}

	return client, nil
}
