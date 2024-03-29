package pkg

import (
	"encoding/base32"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Srtips multiple Ansi from string input
func MultiStripAnsi(str string) string {
	for _, val := range Ansi {
		re := regexp.MustCompile(val)
		str = StripAnsi(str, re)
	}
	return str
}

// StripAnsi ...
func StripAnsi(str string, re *regexp.Regexp) string {
	return re.ReplaceAllString(str, "")
}

func parseDate(dateString string) *time.Time {
	//input := "[2023-06-15T10:38:01Z]"

	// Define the layout that matches the input date format
	layout := "[2006-01-02T15:04:05Z]"

	// Parse the input string to obtain the time.Time value
	t, err := time.Parse(layout, dateString)
	if err != nil {
		fmt.Errorf("Error parsing date:", err)
		return nil
	}

	// fmt.Println("Parsed date:", t)
	return &t
}

func getTestJobRunTime(url, runType string, blobStorage BlobStorage) (*time.Time, error) {
	urlContents, err := downloadTestLog(url, runType, blobStorage)
	if err != nil {
		return nil, err
	}

	contentsByLine := strings.Split(strings.Replace(urlContents, "\r\n", "\n", -1), "\n")

	if len(contentsByLine) == 0 {
		return nil, nil
	}

	// Parse the first line in the file to determine when the test started (and failed.)
	{
		topLine := contentsByLine[0]

		tok1 := strings.Split(topLine, " ")
		if len(tok1) == 0 {
			return nil, nil
		}

		// There's definitely a better way to parse this :P
		tok2 := strings.Split(tok1[0], "/")
		if len(tok2) < 1 {
			return nil, nil
		}

		///
		//input := "\x1b[36mINFO\x1b[0m[2023-06-15T10:38:01Z]"

		// Define the regular expression pattern
		pattern := `\x1b\[\d+m(.*?)\x1b\[\d+m`

		// Compile the regular expression
		re := regexp.MustCompile(pattern)

		// Replace the matched portion with an empty string
		output := re.ReplaceAllString(tok2[0], "")
		// Define the regular expression pattern
		pattern = `\[(.*?)\]`

		// Compile the regular expression
		re = regexp.MustCompile(pattern)

		// Find the first match in the input string
		match := re.FindStringSubmatch(output)

		// fmt.Println("Modified string:", match[1])
		///

		result := parseDate(match[0])
		// result := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.Now().Location())

		return result, nil

	}
}

func parseURL(url, runType string) (string, error) {
	index := strings.LastIndex(url, "/")
	if index == -1 {
		return "", fmt.Errorf("parsing error")
	}

	index = strings.LastIndex(url[0:index-1], "/")
	if index == -1 {
		return "", fmt.Errorf("parsing error")
	}
	index = strings.LastIndex(url[0:index-1], "/")

	if runType == "pull" {
		return "https://storage.googleapis.com/test-platform-results/pr-logs/pull/redhat-developer_gitops-operator" + url[index:] + "/build-log.txt", nil
	} else if runType == "periodic" {
		return "https://storage.googleapis.com/test-platform-results" + url[index:] + "/build-log.txt", nil
	}
	return "https://storage.googleapis.com/test-platform-results" + url[index:] + "/build-log.txt", nil
}

func downloadTestLog(url, runType string, blobStorage BlobStorage) (string, error) {

	value, err := blobStorage.retrieve(url)
	if err != nil {
		return "", err
	}

	if value != "" {
		return value, nil
	}

	// convert
	// https://prow.svc.ci.openshift.org/view/gcs/origin-ci-test/pr-logs/pull/batch/pull-ci-openshift-odo-master-v4.2-integration-e2e-benchmark/2047
	// https://prow.ci.openshift.org/view/gs/origin-ci-test/pr-logs/pull/redhat-developer_odo/5809/pull-ci-redhat-developer-odo-main-v4.10-integration-e2e/1541287908823011328
	// to
	// https://storage.googleapis.com/origin-ci-test/pr-logs/pull/batch/pull-ci-openshift-odo-master-v4.2-integration-e2e-benchmark/2047/build-log.txt
	// https://storage.googleapis.com/origin-ci-test/logs/periodic-ci-openshift-odo-main-v4.8-operatorhub-integration-nightly/1429594453135331328/build-log.txt

	newURL, err := parseURL(url, runType)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", newURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	resp.Body.Close()

	err = blobStorage.store(url, string(byteValue))
	if err != nil {
		return "", err
	}

	return string(byteValue), nil

}

// NewBlobStorage ...
func NewBlobStorage(pathParam string) (*BlobStorage, error) {
	blobStorage := BlobStorage{
		path: pathParam,
	}

	if _, err := os.Stat(pathParam); os.IsNotExist(err) {
		err = os.Mkdir(pathParam, 0755)
		if err != nil {
			return nil, err
		}
	}

	files, err := os.ReadDir(pathParam)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		info, err := os.Stat(pathParam + "/" + f.Name())

		if err != nil {
			return nil, err
		}

		modTime := info.ModTime()

		diff := time.Now().Sub(modTime)

		// Delete cache entries older than 3 weeks
		if diff.Hours() > 24*7*3 {
			err := os.Remove(pathParam + "/" + f.Name())

			if err != nil {
				return nil, err
			}
		}

	}

	return &blobStorage, nil
}

func (s BlobStorage) store(key string, value string) error {
	base64Key := base32.StdEncoding.EncodeToString([]byte(key))
	randIndex := rand.Intn(len(base64Key)/2) + 16
	expectedPath := s.path + "/" + base64Key[randIndex-16:randIndex]

	err := os.WriteFile(expectedPath, []byte(value), 0755)

	return err

}

func (s BlobStorage) retrieve(key string) (string, error) {
	base64Key := base32.StdEncoding.EncodeToString([]byte(key))

	randIndex := rand.Intn(len(base64Key)/2) + 16
	expectedPath := s.path + "/" + base64Key[randIndex-16:randIndex]

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		return "", nil
	}

	contents, err := os.ReadFile(expectedPath)
	if err != nil {
		return "", err
	}

	return string(contents), nil

}
