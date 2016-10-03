package clusterLocal

import "github.com/urfave/cli"

func GetUI() []cli.Command {
	return []cli.Command{
		{
			Name:  "status",
			Usage: "Show cluster status",
			Action: func(c *cli.Context) error {
				printClusterStatus()
				return nil
			},
		},
	}
}
