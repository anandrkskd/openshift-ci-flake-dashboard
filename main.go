package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/anandrkskd/openshift-ci-flake-dashboard/pkg"
)

// method #1 using json struct to store variables
// filename: userconfig.json
func readUserConfig() map[string]interface{} {
	// Open our jsonFile
	jsonFile, err := os.Open("userconfig.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened userconfig.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var userConfig map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &userConfig)
	if err != nil {
		fmt.Println(err)
	}
	return userConfig
}

// # method #2 to use env var files, values seperated by `,`
// filename; userconfig.env
func readUserConfigFromEnvFile() map[string]interface{} {
	readFile, err := os.ReadFile("userconfg.env")
	if err != nil {
		log.Fatal(err)
	}
	envvar := strings.Split(string(readFile), ",")
	userConfig := make(map[string]interface{})
	for _, vals := range envvar {
		userConfig[strings.Trim(strings.Split(vals, "=")[0], "\n")] = strings.Trim(strings.Split(vals, "=")[1], "\n")
	}
	return userConfig
}

func main() {

	// userConfig := readUserConfig()
	userConfig := readUserConfigFromEnvFile()

	// fmt.Println("# odo test statistics")
	// fmt.Println("Generated with https://github.com/jgwest/odo-tools/ and https://github.com/kadel/odo-tools")
	// fmt.Println("## FLAKY TESTS: Failed test scenarios in past 14 days")

	// pkg.PeriodicJobStats(userConfig)
	pkg.PullJobStats(userConfig)
}
