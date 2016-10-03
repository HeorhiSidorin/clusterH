package clusterCommon

import (
	"clusterH/store"
	"fmt"

	"github.com/boltdb/bolt"
)

var db = store.GetDB()

func Status() {
  var clusterName, clusterType string

  db.View(func(tx *bolt.Tx) error {

		bucket := tx.Bucket([]byte("clusterh"))

		clusterName = string(bucket.Get([]byte("currentCluster")))

    clusterType = string(bucket.Get([]byte("currentClusterType")))

		return nil
	})

  if clusterName == "" {
    fmt.Printf("Current context is empty. Create new cluster or switch to existing cluster.\n")
  } else {
    fmt.Printf("Current cluster is %s and called %s\n", clusterType, clusterName)
  }
}
