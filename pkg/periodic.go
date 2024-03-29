package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func PeriodicJobStats(userConfig Config) {
	blobStorage, err := NewBlobStorage("./.cache")
	if err != nil {
		fmt.Println(err)
		return
	}

	testFailMap := map[string]TestFailEntryPriodic{}

	// store search results
	var result Result

	runType := "periodic"
	req, err := http.NewRequest("GET", "https://search.ci.openshift.org/search", nil)
	if err != nil {
		panic(err)
	}

	// https://search.ci.openshift.org/search?context=0&maxAge=336h&maxBytes=20971520&maxMatches=5&name=pull-ci-openshift-odo-main-&search=%5C%5BFail%5C%5D&type=build-log
	q := req.URL.Query()
	q.Add("search", fmt.Sprintf("%s", userConfig.SearchStr))
	q.Add("maxAge", "336h")
	q.Add("context", "0")
	q.Add("type", "build-log")
	q.Add("name", fmt.Sprintf("periodic-ci-%s-%s-master-", userConfig.RepoOrg, userConfig.RepoName))
	q.Add("maxMatches", "5")
	q.Add("maxBytes", "20971520")
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		panic(err)
	}

	// fmt.Println("map:", string(byteValue))

	// iterate over all results
	for k, search := range result {
		if strings.Contains(k, "rehearse") {
			break
		}
		expectedBuildLogURL, err := parseURL(k, runType)
		if err != nil {
			expectedBuildLogURL = ""
		}

		runTime, err := getTestJobRunTime(k, runType, *blobStorage)
		if err != nil {
			fmt.Printf("Error occurred on test log download: %v ", err)
			return
		}

		StrIndex := strings.Index(k, fmt.Sprintf("%s-%s", userConfig.RepoOrg, userConfig.RepoName))

		var prNumber string
		if StrIndex > -1 {
			str := k[StrIndex:]
			strArr := strings.Split(str, "-")
			// fmt.Print(strArr, "\n")
			prNumber = strArr[5]
		}

		// fmt.Printf("%s\n", file)
		// fmt.Println("map:", search)
		for _, matches := range search {
			// fmt.Printf("  %v\n", regexp)
			for _, match := range matches {
				lines := []string{}
				for _, line := range match.Context {
					// fmt.Printf("    %v\n", line)
					cleanLine := strings.TrimSpace(line)
					cleanLine = MultiStripAnsi(cleanLine)
					// de-duplication
					// count each line only once
					dup := false
					for _, l := range lines {
						if l == cleanLine {
							dup = true
						}
					}
					if !dup {

						entry, exists := testFailMap[cleanLine]
						if !exists {
							entry = TestFailEntryPriodic{LogURLs: map[string][]string{}}
						}

						entry.TestFail++

						lines = append(lines, cleanLine)

						if runTime != nil {

							val := entry.LastSeen
							if val == nil {
								val = runTime
							}

							if runTime.After(*val) {
								val = runTime
							}
							entry.LastSeen = val

						}

						if prNumber != "" {

							matchFound := false
							for _, existingEntry := range entry.PRList {
								if existingEntry == prNumber {
									matchFound = true
								}
							}

							if !matchFound {
								entry.PRList = append(entry.PRList, prNumber)
							}

							// Add build log URL for the PR
							logURLList := entry.LogURLs[prNumber]
							logURLList = append(logURLList, expectedBuildLogURL)
							entry.LogURLs[prNumber] = logURLList
							//fmt.Println(entry.LogURLs, "\n")
						}

						testFailMap[cleanLine] = entry

					}
				}
			}
		}

	}

	type TestFails struct {
		Score    int
		TestName string
		Fails    int
		LastSeen string
		PRList   []string
		Entry    TestFailEntryPriodic
	}

	// convert tests to slice so we can easily sort it
	fails := []TestFails{}
	for test, entry := range testFailMap {

		prList := entry.PRList
		lastSeenVal := ""

		// Score calculation
		score := 0
		{
			daysSinceLastSeen := 1

			lastSeenTime := entry.LastSeen
			if lastSeenTime != nil {

				//days := time.Now().Sub(*lastSeenTime).Hours() / 24
				days := time.Since(*lastSeenTime).Hours() / 24

				lastSeenVal = fmt.Sprintf("%d days ago", int(days))

				daysSinceLastSeen = int(days)
			}

			if daysSinceLastSeen == 0 {
				daysSinceLastSeen = 1
			}

			prListSize := len(prList)

			if prListSize > 6 {
				// >6 PRs does not imply any further strength than 6 PRs, for score calculation purposes.
				prListSize = 6
			}

			score = (10 * prListSize * entry.TestFail) / (daysSinceLastSeen)

			// Minimum score if there is at least one PR, and at least one fail, is 1
			if score == 0 && len(prList) > 0 && entry.TestFail > 0 {
				score = 1
			}
		}

		fails = append(fails, TestFails{TestName: test, Fails: entry.TestFail, PRList: prList, Score: score, LastSeen: lastSeenVal, Entry: entry})
	}

	if len(fails) == 0 {
		fmt.Println("\n### *No Test failures found for last 14 days of __Periodic__ test runs*")
		return
	}

	sort.Slice(fails, func(i, j int) bool {
		one := fails[i].Score
		two := fails[j].Score

		// Primary sort: descending by score
		if one != two {
			return one > two
		}

		// Secondary sort: descending by fails
		one = fails[i].Fails
		two = fails[j].Fails
		if one != two {
			return one > two
		}

		// Tertiary sort: decreasing by pr list size
		one = len(fails[i].PRList)
		two = len(fails[j].PRList)
		if one != two {
			return one > two
		}

		// Finally, sort ascending by name
		return fails[j].TestName > fails[i].TestName
	})

	fmt.Println("## FLAKY TESTS: Failed test scenarios in past 14 days")
	fmt.Println("| Failure Score<sup>*</sup> | Failures | Test Name | Last Seen | PR List and Logs ")
	fmt.Println("|---|---|---|---|---|")
	for _, f := range fails {

		prListString := fmt.Sprintf("%d: ", len(f.PRList))
		for _, prNumber := range f.PRList {

			logURLs := f.Entry.LogURLs[prNumber]

			prListString += fmt.Sprintf("[%s]", prNumber)

			if len(logURLs) > 0 {

				prListString += "<sup>"

				for index, logURL := range logURLs {
					prListString += "[" + strconv.FormatInt(int64(index+1), 10) + "](" + logURL + ")"

					if index+1 != len(logURLs) {
						prListString += ", "
					}
				}

				prListString += "</sup>"
			}

			prListString += " "
		}

		fmt.Printf("| %d | %d | %s | %s | %s\n", f.Score, f.Fails, f.TestName, f.LastSeen, prListString)
	}
}
