package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"openshift-ci-flake-dashboard/pkg"
	"os"
	"strings"
)

// method #1 using json struct to store variables
// filename: userconfig.json
func readUserConfig(config *pkg.Config) {
	// Open our jsonFile
	jsonFile, err := os.Open("userconfig.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		fmt.Println(err)
	}
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

	var userConfig pkg.Config
	readUserConfig(&userConfig)
	//userConfig := readUserConfigFromEnvFile()

	pkg.Ansi = append(pkg.Ansi, userConfig.Regex)
	//fmt.Printf("# test config %v \n", userConfig)

	// fmt.Println("Generated with https://github.com/jgwest/odo-tools/ and https://github.com/kadel/odo-tools")
	// fmt.Println("## FLAKY TESTS: Failed test scenarios in past 14 days")
	//
	pkg.PullJobStats(userConfig)
	pkg.PeriodicJobStats(userConfig)
}
