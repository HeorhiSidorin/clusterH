package clusterDO

import (
	"bufio"
	"clusterH/store"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/digitalocean/godo"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
)

var db = store.GetDB()

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func createDoCluster(c *cli.Context) error {

	//  id, _ := uuid.NewV4()
	number := c.Int("number")
	region := c.String("region")
	size := c.String("size")
	pat := c.String("token")
	filePath := c.String("file")

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	// create cluster's tag
	tagCreateRequest := &godo.TagCreateRequest{
		Name: c.String("name"),
	}
	client.Tags.Create(tagCreateRequest)

	var names []string
	names = make([]string, number, number)
	for i := 0; i < number; i++ {
		names[i] = "mcine-" + strconv.Itoa(i)
	}

	//open user-data file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	scanner := bufio.NewScanner(file)

	var userDataLinesArray []string

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		userDataLinesArray = append(userDataLinesArray, line)
	}

	userData := strings.Join(userDataLinesArray, "")

	createRequest := &godo.DropletMultiCreateRequest{
		Names:             names,
		Region:            region,
		Size:              size,
		PrivateNetworking: true,
		SSHKeys: []godo.DropletCreateSSHKey{
			{
				Fingerprint: "d2:ee:f3:b0:a4:de:95:12:4c:27:24:5f:de:bb:87:90",
			},
		},
		Image: godo.DropletCreateImage{
			Slug: "coreos-stable",
		},
		UserData: userData,
	}

	droplets, _, err := client.Droplets.CreateMultiple(createRequest)

	if err != nil {
		fmt.Printf("Something bad happened: %s\n\n", err)
		return err
	}

	// tagging cluster's droplets
	var resources []godo.Resource
	for _, droplet := range droplets {
		resources = append(resources, godo.Resource{
			ID:   strconv.Itoa(droplet.ID),
			Type: "droplet",
		})
	}

	tagResourcesRequest := &godo.TagResourcesRequest{
		Resources: resources,
	}
	client.Tags.TagResources(c.String("name"), tagResourcesRequest)

	//removing first droplet from cluster(untagging droplet)
	unTagResourcesRequest := &godo.UntagResourcesRequest{
		Resources: resources[0:1],
	}

	client.Tags.UntagResources(c.String("name"), unTagResourcesRequest)

	return nil
}

func DestroyAll(c *cli.Context) {
	pat := c.String("token")

	tokenSource := &TokenSource{
		AccessToken: pat,
	}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	droplets, _, _ := client.Droplets.List(&godo.ListOptions{})

	for _, d := range droplets {
		fmt.Println(d.ID)
		client.Droplets.Delete(d.ID)
	}
}

func AddFingerprint(fingerprint, name string) error {

	var nameExist = false

	db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte("fingerprints"))

		if bucket == nil {
			return nil
		}

		if bucket.Get([]byte(name)) != nil {
			nameExist = true
		}
		return nil
	})

	if nameExist {
		fmt.Println("This name of fingerprint is already exist. Please choose other name")
		return nil
	}

	db.Update(func(tx *bolt.Tx) error {

		bucket, _ := tx.CreateBucketIfNotExists([]byte("fingerprints"))

		var key = []byte(name)
		var value = []byte(fingerprint)

		bucket.Put(key, value)

		return nil
	})

	return nil
}

func Create(c *cli.Context) error {

	var clusterName = c.String("name")
	var bucketName = []byte("clusterh")

	var clusters []string

	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		if bucket == nil {
			return nil
		}

		json.Unmarshal(bucket.Get([]byte("clusters")), &clusters)

		return nil
	})

	for _, v := range clusters {
		if v == clusterName {
			fmt.Printf("%s is already used as cluster name. Please choose another. \n", clusterName)
			return nil
		}
	}

	clusters = append(clusters, string(clusterName))

	db.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists(bucketName)

		_ = bucket.Put([]byte("currentCluster"), []byte(clusterName))

		_ = bucket.Put([]byte("currentClusterType"), []byte("do"))

		clusters, _ := json.Marshal(clusters)
		_ = bucket.Put([]byte("clusters"), clusters)

		_ = bucket.Put([]byte(clusterName+"-token"), []byte(c.String("token")))

		return nil
	})

	createDoCluster(c)

	return nil
}

func Fingerprint(c *cli.Context) {
	db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte("fingerprints"))

		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()

	  for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
	    fmt.Printf("%s --------> %s\n", k, v)
	  }

	  return nil
	})
}

func GetUI() []cli.Command {
	return []cli.Command{
		{
			Name:  "destroy",
			Usage: "destroy all droplets in account",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "token, t",
					Value:  "edb76f943aed64b72856bf99de5ce1608284fbedcf76ec32491ee19c566be7e2",
					Usage:  "Your digitalocean's token",
					EnvVar: "DIGITAL_OCEAN_TOKEN",
				},
			},
			Action: func(c *cli.Context) error {
				DestroyAll(c)
				return nil
			},
		},
	}
}
