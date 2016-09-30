package clusterOperation

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

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

func createLocalClusterConfigs(cliFlags *cli.Context, configPath string) error {
	file, err := os.OpenFile(configPath+"/config.rb.sample", os.O_RDONLY, 0777)
	data := bytes.NewBuffer(nil)

	io.Copy(data, file)

	file.Close()

	sampleConfig := string(data.Bytes())

	fmt.Println(strings.Index(sampleConfig, "$num_instances="))

	config := strings.Replace(sampleConfig, "$num_instances=1", "$num_instances="+cliFlags.String("number"), 1)

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
	runningFilesDest := currentUser.HomeDir + "/.config/clusterH"
	runningFilesPath := runningFilesDest + "/coreos-vagrant-master"

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
	err = createLocalClusterConfigs(c, runningFilesPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Run cluster in vagrant")
	runVagrant(runningFilesPath)
	fmt.Printf("You can use plain vagrant commands in such directory: %s \n", runningFilesPath)

}
