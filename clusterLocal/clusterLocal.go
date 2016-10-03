package clusterLocal

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"clusterH/store"

	"github.com/boltdb/bolt"
	"github.com/urfave/cli"
)

var db = store.GetDB()

type localClusterConfig struct {
	NumInstances        uint
	InstanceNamePrefix  string
	UpdateChannel       string
	ImageVersion        string
	EnableSerialLogging bool
	ShareHome           bool
	VMGui               bool
	VMMemory            uint
	VMCpus              uint
	VbCPUExecutionCap   uint
	SharedFolders       map[string]string
	ForwardedPorts      map[string]string
}

var rubyVagrantConfig = `
require 'json'

clusterConfig = JSON.parse(File.read('../config.json'))

$num_instances = clusterConfig['NumInstances']
$instance_name_prefix = clusterConfig['InstanceNamePrefix']
$update_channel = clusterConfig['UpdateChannel']
$image_version = clusterConfig['ImageVersion']
$enable_serial_logging = clusterConfig['EnableSerialLogging']
$share_home = clusterConfig['ShareHome']
$vm_gui = clusterConfig['VMGui']
$vm_memory = clusterConfig['VMMemory']
$vm_cpus = clusterConfig['VMCpus']
$vb_cpuexecutioncap = clusterConfig['VbCPUExecutionCap']
$shared_folders = clusterConfig['SharedFolders']
$forwarded_ports = clusterConfig['ForwardedPorts']
`

func Init() {

}

func downloadRunningFiles(filepath, url string) error {
	usr, _ := user.Current()
	out, err := os.Create(usr.HomeDir + "/.config/clusterH/coreos-vagrant.zip")
	if err != nil {
		return err
	}

	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}

	return nil
}

func unzipRunningFiles(archivePath, target string) error {
	reader, err := zip.OpenReader(archivePath)

	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	return nil
}

func createLocalClusterConfigs(cliFlags *cli.Context, configPath string, jsonConfigPath string) error {

	myConfig := localClusterConfig{
		cliFlags.Uint("number"),
		cliFlags.String("name"),
		"alpha",
		"current",
		false,
		false,
		false,
		1024,
		1,
		100,
		map[string]string{},
		map[string]string{},
	}

	myConfigJSON, _ := json.Marshal(myConfig)
	myConfigJSONFile, err := os.OpenFile(jsonConfigPath+"/config.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer myConfigJSONFile.Close()

	myConfigJSONFile.Write([]byte(myConfigJSON))

	file, err := os.OpenFile(configPath+"/config.rb.sample", os.O_RDONLY, 0777)
	data := bytes.NewBuffer(nil)

	io.Copy(data, file)

	file.Close()

	sampleConfig := string(data.Bytes())

	config := sampleConfig + rubyVagrantConfig

	configFile, err := os.OpenFile(configPath+"/config.rb", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer configFile.Close()

	_, err = configFile.Write([]byte(config))
	if err != nil {
		return err
	}

	userDataSample, err := os.Open(configPath + "/user-data.sample")
	if err != nil {
		return err
	}
	defer userDataSample.Close()

	userData, err := os.Create(configPath + "/user-data")
	if err != nil {
		return err
	}
	defer userData.Close()

	_, err = io.Copy(userData, userDataSample)
	if err != nil {
		return err
	}

	return nil
}

func runVagrant(workingDir string) {
	oldWorkingDir, _ := os.Getwd()
	os.Chdir(workingDir)
	defer os.Chdir(oldWorkingDir)
	cmd := exec.Command("vagrant", "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func createLocalCluster(c *cli.Context) {
	currentUser, _ := user.Current()
	runningArchiveURL := "https://github.com/coreos/coreos-vagrant/archive/master.zip"
	runningArchivePath := currentUser.HomeDir + "/.config/clusterH/coreos-vagrant.zip"
	runningFilesDest := currentUser.HomeDir + "/.config/clusterH/" + c.String("name")
	workDir := runningFilesDest + "/coreos-vagrant-master"

	fmt.Println("Downloading coreos-vagrant files")
	err := downloadRunningFiles(runningArchivePath, runningArchiveURL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Unzip downloaded archive")
	err = unzipRunningFiles(runningArchivePath, runningFilesDest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Create cluster confings: config.rb and user-data")
	err = createLocalClusterConfigs(c, workDir, runningFilesDest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Run cluster in vagrant")
	runVagrant(workDir)
	fmt.Printf("You can use plain vagrant commands in such directory: %s \n", workDir)

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
		bucket, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		_ = bucket.Put([]byte("currentCluster"), []byte(clusterName))

		_ = bucket.Put([]byte("currentClusterType"), []byte("local"))

		clusters, err := json.Marshal(clusters)
		if err != nil {
			return err
		}
		_ = bucket.Put([]byte("clusters"), clusters)

		_ = bucket.Put([]byte(clusterName+"-token"), []byte(c.String("token")))

		return nil
	})

	createLocalCluster(c)

	return nil
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

func printClusterStatus() {
	fmt.Println(getCurrentCluster())
	var workDir string
	currentUser, _ := user.Current()
	db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("clusterh"))

		if bucket == nil {
			return nil
		}

		workDir = currentUser.HomeDir + "/.config/clusterH/" + string(bucket.Get([]byte("currentCluster"))) + "/coreos-vagrant-master"

		return nil
	})
	fmt.Println(workDir)
	oldWorkingDir, _ := os.Getwd()
	os.Chdir(workDir)
	defer os.Chdir(oldWorkingDir)
	cmd := exec.Command("vagrant", "status")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
