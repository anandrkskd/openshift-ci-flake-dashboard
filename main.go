package main

import (
	"fmt"

	"github.com/anandrkskd/openshift-ci-flake-dashboard/pkg"
)

func main() {

	fmt.Println("# odo test statistics")
	fmt.Println("Generated with https://github.com/jgwest/odo-tools/ and https://github.com/kadel/odo-tools")
	fmt.Println("## FLAKY TESTS: Failed test scenarios in past 14 days")

	pkg.PeriodicJobStats()
	pkg.PullJobStats()
}
