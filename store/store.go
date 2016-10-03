package store

import (
	"log"
	"os"
	"os/user"

	"github.com/boltdb/bolt"
)

var db *bolt.DB

func init() {
	usr, _ := user.Current()

	if _, err := os.Stat(usr.HomeDir + "/.config/clusterH"); os.IsNotExist(err) {
		os.Mkdir(usr.HomeDir+"/.config/clusterH", 0700)
	}

	var err error
	db, err = bolt.Open(usr.HomeDir+"/.config/clusterH/clusterH.db", 0644, nil)

	if err != nil {
		log.Fatal(err)
	}
}

func Close() {
	db.Close()
}

func GetDB() *bolt.DB {
	return db
}

func GetCurrentClusterType() string {
	var currentClusterType string

	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("clusterh"))

		if bucket == nil {
			return nil
		}

		currentClusterType = string(bucket.Get([]byte("currentClusterType")))

		return nil
	})

	return currentClusterType
}
