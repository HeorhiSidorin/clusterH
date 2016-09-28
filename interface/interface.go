package interface;

import (
	"fmt"
	"os"
	"strconv"

	"github.com/digitalocean/godo"
	"github.com/nu7hatch/gouuid"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func Run() {
	id, err := uuid.NewV4()
	_ = err
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "deploy",
			Usage: "deploy cluster of CoreOS machines",
			Subcommands: []cli.Command{
				{
					Name: "do",
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
							Value: id.String(),
							Usage: "number of machines in cluster",
						},
					},
					Usage: "deploy on digitalocean",
					Action: func(c *cli.Context) error {
						pat := c.String("token")

						tokenSource := &TokenSource{
							AccessToken: pat,
						}
						oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
						client := godo.NewClient(oauthClient)

						var names []string
						names = make([]string, c.Int("number"), c.Int("number"))
						for i := 0; i < c.Int("number"); i++ {
							names[i] = "mcine-" + strconv.Itoa(i)
						}
						createRequest := &godo.DropletMultiCreateRequest{
							Names:  names,
							Region: "nyc3",
							Size:   "512mb",
							SSHKeys: []godo.DropletCreateSSHKey{
								{
									Fingerprint: "0e:4e:20:87:d6:fd:9d:a1:bb:32:33:0c:cd:e3:d0:c7",
								},
							},
							Image: godo.DropletCreateImage{
								Slug: "coreos-stable",
							},
						}

						droplets, _, err := client.Droplets.CreateMultiple(createRequest)

						if err != nil {
							fmt.Printf("Something bad happened: %s\n\n", err)
							return err
						}

						fmt.Printf(droplets[0].PublicIPv4())
						return nil
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
