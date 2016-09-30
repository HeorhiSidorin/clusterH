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

// func getInfo(c *cli.Context) {
// 	pat := c.String("token")
//
// 	tokenSource := &TokenSource{
// 		AccessToken: pat,
// 	}
// 	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
// 	client := godo.NewClient(oauthClient)
// 	//
// 	opt := &godo.ListOptions{
// 		Page:    1,
// 		PerPage: 2,
// 	}
//
// 	droplets, _, _ := client.Droplets.List(opt)
// 	//
// 	fmt.Println(droplets[0].ID)
//
// 	// Open the file.
// 	f, _ := os.Open("/home/heorhi/cli/src/clusterH/clusterOperation/user-data")
// 	// Create a new Scanner for the file.
// 	scanner := bufio.NewScanner(f)
// 	// Loop over all lines in the file and print them.
//
// 	var userDataLinesArray []string
//
// 	for scanner.Scan() {
// 		line := scanner.Text() + "\n"
// 		userDataLinesArray = append(userDataLinesArray, line)
// 	}
//
// 	fmt.Println(strings.Join(userDataLinesArray, ""))
// }

func createDoCluster(c *cli.Context) error {

	//  id, _ := uuid.NewV4()
	number := c.Int("number")
	region := c.String("region")
	size := c.String("size")
	pat := c.String("token")

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
	file, err := os.Open(c.Args()[0])
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
				Fingerprint: "0e:4e:20:87:d6:fd:9d:a1:bb:32:33:0c:cd:e3:d0:c7",
			},
			{
				Fingerprint: "86:36:52:f1:9b:35:fc:d9:fe:17:a9:67:99:5d:74:39",
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

	// // store ip addresses of cluster's members
	// err = db.Update(func(tx *bolt.Tx) error {
	// 	bucket, _ := tx.CreateBucketIfNotExists(newClusterName)
	//
	// 	key := []byte("members")
	// 	stringByte := "\x00" + strings.Join(ipAdresses, "\x20\x00")
	// 	value := []byte(stringByte)
	//
	// 	err = bucket.Put(key, value)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// })
	//
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// // store ip addresses of cluster's members
	// err = db.Update(func(tx *bolt.Tx) error {
	// 	bucket, _ := tx.CreateBucketIfNotExists(contextBucket)
	//
	// 	key := []byte("currentContext")
	// 	value := []byte(newClusterName)
	//
	// 	err = bucket.Put(key, value)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// })
	//
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// //retrieve current context
	// err = db.View(func(tx *bolt.Tx) error {
	// 	bucket := tx.Bucket(contextBucket)
	// 	if bucket == nil {
	// 		return fmt.Errorf("Bucket %q not found!", contextBucket)
	// 	}
	//
	// 	key := []byte("currentContext")
	//
	// 	val := bucket.Get(key)
	// 	fmt.Println(string(val))
	//
	// 	return nil
	// })
	//
	// if err != nil {
	// 	log.Fatal(err)
	// }

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

func GetUI() []cli.Command {
	return []cli.Command{
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
				DestroyAll(c)
				return nil
			},
		},
	}
}
