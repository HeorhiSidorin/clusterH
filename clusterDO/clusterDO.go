package clusterDO

import (
	"bufio"
	"clusterH/store"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/digitalocean/godo"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
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

	userDataYAML := make(map[interface{}]interface{})

	_ = yaml.Unmarshal([]byte(userData), &userDataYAML)

	//generate new discovery
	resp, _ := http.Get("https://discovery.etcd.io/new?size=" + strconv.Itoa(number))
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	userDataYAML["coreos"].(map[interface{}]interface{})["etcd2"].(map[interface{}]interface{})["discovery"] = string(body)

	userDataBytes, _ := yaml.Marshal(&userDataYAML)

	userData = string(userDataBytes)

	createRequest := &godo.DropletMultiCreateRequest{
		Names:             names,
		Region:            region,
		Size:              size,
		PrivateNetworking: true,
		SSHKeys: []godo.DropletCreateSSHKey{
			{
				Fingerprint: "d2:ee:f3:b0:a4:de:95:12:4c:27:24:5f:de:bb:87:90",
			},
			{
				Fingerprint: "0e:4e:20:87:d6:fd:9d:a1:bb:32:33:0c:cd:e3:d0:c7",
			},
			{
				Fingerprint: "9d:ee:ae:a0:56:42:73:b5:95:4e:f2:ff:54:d4:1a:78",
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
		fmt.Println(droplet.Status)
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

func Destroy(c *cli.Context) {
	var pat string
	var clusters []string
	var clusterName string
	var clusterIndex int

	db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte("clusterh"))

		json.Unmarshal(bucket.Get([]byte("clusters")), &clusters)

		clusterName = string(bucket.Get([]byte("currentCluster")))

		pat = string(bucket.Get([]byte(clusterName + "-token")))

		return nil
	})

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

	for i, c := range clusters {
		if c == clusterName {
			clusterIndex = i
			break
		}
	}

	clusters = append(clusters[:clusterIndex], clusters[clusterIndex+1:]...)

	db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("clusterh"))

		_ = bucket.Put([]byte("currentCluster"), []byte(""))

		_ = bucket.Put([]byte("currentClusterType"), []byte(""))

		clusters, _ := json.Marshal(clusters)
		_ = bucket.Put([]byte("clusters"), clusters)

		_ = bucket.Delete([]byte(clusterName + "-token"))

		return nil
	})
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

func getCurrentCluster() struct{ cName, cType string } {
	var currentClusterName, currentClusterType string

	db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte("clusterh"))

		currentClusterName = string(bucket.Get([]byte("currentCluster")))

		currentClusterType = string(bucket.Get([]byte("currentClusterType")))

		return nil
	})

	return struct{ cName, cType string }{
		cName: currentClusterName,
		cType: currentClusterType,
	}
}

func status() {
	cluster := getCurrentCluster()
	fmt.Println(cluster)
}

func GetUI() []cli.Command {
	return []cli.Command{
		{
			Name:  "destroy",
			Usage: "destroy all droplets in account",
			Action: func(c *cli.Context) error {
				Destroy(c)
				return nil
			},
		},
		{
			Name:  "status",
			Usage: "status of clusterH",
			Action: func(c *cli.Context) error {
				status()
				return nil
			},
		},
	}
}
