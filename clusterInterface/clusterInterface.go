package clusterInterface

import (
	"fmt"
	"os"
	"strings"

	"clusterH/clusterOperation"

	"github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

func Run() {
	name := uuid.NewV4().String()
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "create cluster of CoreOS machines",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "token, t",
					Value: "6300f115ed7a9c6c3d5f334e8e511637841a55ceb1f45ab692592c755419d0fd",
					Usage: "your digitalocean's token (required)",
				},
				cli.StringFlag{
					Name: "region",
					Usage: `new cluster region
	List of regions:
	nyc1 - New York 1
	nyc2
	nyc3
	sfo1 - San Francisco 1
	sfo2
	ams2 - Amsterdam 2
	ams3
	sgp1 - Singapore
	lon1 - London
	fra1 - Frankfurt
	tor1 - Toronto
	blr1 - Bangalore`,
					Value: "fra1",
				},
				cli.IntFlag{
					Name:  "number, n",
					Usage: "number of machines in new cluster (required)",
				},
				cli.StringFlag{
					Name:  "name",
					Value: name,
					Usage: "new cluster name (required)",
				},
				cli.StringFlag{
					Name:  "type",
					Usage: "new cluster type (required)",
				},
				cli.StringFlag{
					Name: "size",
					Usage: `new cluster size
	List of sizes: 512mb, 1gb, 2gb, 4gb, 8gb, 16gb`,
					Value: "512mb",
				},
			},
			Action: func(c *cli.Context) error {
				//check flags count
				if c.NumFlags() < 3 {
					fmt.Println("Incorrect command usage. See CREATE command help.")
					cli.ShowCommandHelp(c, "create")
					return nil
				}
				clusterOperation.Create(c)
				return nil
			},
		},
		{
			Name:  "add",
			Usage: "command for addig something",
			Subcommands: []cli.Command{
				{
					Name:  "fingerprint",
					Usage: "add fingerprint to database",
					Action: func(c *cli.Context) error {
						if len(c.Args()[0]) != 47 && strings.Count(c.Args()[0], ":") != 15 {
							fmt.Printf("Wrong command's format.\nclusterH add fingerprint <fingerprint> <name>\n")
							return nil
						}
						clusterOperation.AddFingerprint(c.Args()[0], c.Args()[1])
						return nil
					},
				},
			},
		},
		{
			Name:  "destroy",
			Usage: "destroy all droplets in account",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "token, t",
					Value:  "6300f115ed7a9c6c3d5f334e8e511637841a55ceb1f45ab692592c755419d0fd",
					Usage:  "Your digitalocean's token",
					EnvVar: "DIGITAL_OCEAN_TOKEN",
				},
			},
			Action: func(c *cli.Context) error {
				clusterOperation.DestroyAll(c)
				return nil
			},
		},
	}
	app.Run(os.Args)
}
