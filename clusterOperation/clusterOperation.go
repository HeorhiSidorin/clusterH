package clusterOperation

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/boltdb/bolt"
	"github.com/urfave/cli"
)

func AddFingerprint(fingerprint, name string) error {

	usr, _ := user.Current()

	if _, err := os.Stat(usr.HomeDir + "/.config/clusterH"); os.IsNotExist(err) {
		os.Mkdir(usr.HomeDir+"/.config/clusterH", 0700)
	}

	db, err := bolt.Open(usr.HomeDir+"/.config/clusterH/clusterH.db", 0644, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var nameExist bool = false

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

	usr, _ := user.Current()

	if _, err := os.Stat(usr.HomeDir + "/.config/clusterH"); os.IsNotExist(err) {
		os.Mkdir(usr.HomeDir+"/.config/clusterH", 0700)
	}

	db, err := bolt.Open(usr.HomeDir+"/.config/clusterH/clusterH.db", 0644, nil)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

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

		clusters, _ := json.Marshal(clusters)
		_ = bucket.Put([]byte("clusters"), clusters)

		_ = bucket.Put([]byte(clusterName+"-token"), []byte(c.String("token")))

		return nil
	})

	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName)

		var clusters []string
		json.Unmarshal(bucket.Get([]byte("clusters")), &clusters)
		fmt.Println("!!!!----->>", clusters)
		fmt.Println("$$$$----->>", string(bucket.Get([]byte("currentCluster"))))
		fmt.Println(string(bucket.Get([]byte(clusterName + "-token"))))

		return nil
	})

	if c.String("type") == "do" {
		createDoCluster(c)
	} else if c.String("type") == "local" {
		createLocalCluster(c)
	} else {
		fmt.Printf("Please choose cluster type: local or do")
		return nil
	}

	return nil
}
