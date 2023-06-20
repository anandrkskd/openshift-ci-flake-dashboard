package pkg

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var runType string = "periodic"

func PeriodicJobStats() {
	blobStorage, err := NewBlobStorage("./.cache")
	if err != nil {
		fmt.Println(err)
		return
	}

	result, err := getSearchResults()
	if err != nil {
		fmt.Println("Error occurred while fetching search results:", err)
		return
	}

	testFailMap := processSearchResults(result, blobStorage)

	fails := convertTestFailMapToSlice(testFailMap)

	sortFailsByScore(fails)

	printTestStatistics(fails)
}

func getSearchResults() (Result, error) {
	// runType := "periodic"
	req, err := http.NewRequest("GET", "https://search.ci.openshift.org/search", nil)
	if err != nil {
		return Result{}, err
	}

	q := req.URL.Query()
	q.Add("search", "\\[FAIL\\]")
	q.Add("maxAge", "488h")
	q.Add("context", "0")
	q.Add("type", "build-log")
	q.Add("name", "periodic-ci-redhat-developer-odo-main-")
	q.Add("maxMatches", "5")
	q.Add("maxBytes", "20971520")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	byteValue, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}

	var result Result
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return Result{}, err
	}

	return result, nil
}

func processSearchResults(result Result, blobStorage *BlobStorage) map[string]TestFailEntry {
	testFailMap := make(map[string]TestFailEntry)

	for k, search := range result {
		expectedBuildLogURL, err := parseURL(k, runType)
		if err != nil {
			expectedBuildLogURL = ""
		}

		runTime, err := getTestJobRunTime(k, runType, *blobStorage)
		if err != nil {
			fmt.Printf("Error occurred on test log download: %v ", err)
			continue
		}

		odoIndex := strings.Index(k, "redhat-developer-odo-main-")
		var prNumber string
		if odoIndex > -1 {
			str := k[odoIndex:]
			strArr := strings.Split(str, "-")
			prNumber = strArr[4]
		}

		re := regexp.MustCompile(ansi)

		for _, matches := range search {
			for _, match := range matches {
				lines := []string{}
				for _, line := range match.Context {
					cleanLine := strings.TrimSpace(line)
					cleanLine = StripAnsi(cleanLine, re)
					dup := false
					for _, l := range lines {
						if l == cleanLine {
							dup = true
						}
					}
					if !dup {
						entry, exists := testFailMap[cleanLine]
						if !exists {
							entry = TestFailEntry{LogURLs: map[any][]string{}}
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

							logURLList := entry.LogURLs[prNumber]
							logURLList = append(logURLList, expectedBuildLogURL)
							entry.LogURLs[prNumber] = logURLList
						}

						testFailMap[cleanLine] = entry
					}
				}
			}
		}
	}

	return testFailMap
}

func convertTestFailMapToSlice(testFailMap map[string]TestFailEntry) []TestFails {
	fails := []TestFails{}
	for test, entry := range testFailMap {
		prList := entry.PRList
		lastSeenVal := ""

		score := 0
		{
			daysSinceLastSeen := 1
			lastSeenTime := entry.LastSeen
			if lastSeenTime != nil {
				days := time.Since(*lastSeenTime).Hours() / 24
				lastSeenVal = fmt.Sprintf("%d days ago", int(days))
				daysSinceLastSeen = int(days)
			}

			if daysSinceLastSeen == 0 {
				daysSinceLastSeen = 1
			}

			prListSize := len(prList)

			if prListSize > 6 {
				prListSize = 6
			}

			score = (10 * prListSize * entry.TestFail) / (daysSinceLastSeen)

			if score == 0 && len(prList) > 0 && entry.TestFail > 0 {
				score = 1
			}
		}

		fails = append(fails, TestFails{TestName: test, Fails: entry.TestFail, PRList: prList, Score: score, LastSeen: lastSeenVal, Entry: entry})
	}

	return fails
}

func sortFailsByScore(fails []TestFails) {
	sort.Slice(fails, func(i, j int) bool {
		one := fails[i].Score
		two := fails[j].Score

		if one != two {
			return one > two
		}

		one = fails[i].Fails
		two = fails[j].Fails
		if one != two {
			return one > two
		}

		one = len(fails[i].PRList)
		two = len(fails[j].PRList)
		if one != two {
			return one > two
		}

		return fails[j].TestName > fails[i].TestName
	})
}

func printTestStatistics(fails []TestFails) {
	fmt.Println("# odo test statistics for periodic jobs")
	fmt.Printf("Last update: %s (UTC)\n\n", time.Now().UTC().Format("2006-01-02 15:04:05"))
	fmt.Println("| Failure Score<sup>*</sup> | Failures | Test Name | Last Seen | Cluster version and Logs ")
	fmt.Println("|---|---|---|---|---|")
	for _, f := range fails {
		if len(f.PRList) <= 1 {
			continue
		}

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

	fmt.Println()
	fmt.Println()
}
