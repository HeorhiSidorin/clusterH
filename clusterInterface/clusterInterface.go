package clusterInterface

import (
	"os"

	"clusterH/clusterOperation"

	"github.com/urfave/cli"
)

func Run() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "create cluster of CoreOS machines",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "token, t",
					Value:  "6300f115ed7a9c6c3d5f334e8e511637841a55ceb1f45ab692592c755419d0fd",
					Usage:  "Your digitalocean's token",
					EnvVar: "DIGITAL_OCEAN_TOKEN",
				},
				cli.IntFlag{
					Name:  "number, n",
					Value: 3,
					Usage: "number of machines in cluster",
				},
				cli.StringFlag{
					Name:  "cluster-name",
					Value: "fisrt",
					Usage: "number of machines in cluster",
				},
				cli.StringFlag{
					Name:  "type",
					Value: "do",
					Usage: "number of machines in cluster",
				},
			},
			Action: func(c *cli.Context) error {
				clusterOperation.Create(c)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
