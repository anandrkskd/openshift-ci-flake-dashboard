package pkg

import (
	"regexp"
	"time"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

const ansiTime = `\(\d+\.\d+s\)`
const ansiPrefix = `---\s+FAIL:\s+kuttl/harness/`

// StripAnsi ...
func StripAnsi(str string, re *regexp.Regexp) string {
	return re.ReplaceAllString(str, "")
}

// Match ...
type Match struct {
	FileType  string   `json:"filename"`
	Context   []string `json:"context,omitempty"`
	MoreLines int      `json:"moreLines,omitempty"`
}

// TestFailEntry ...
type TestFailEntry struct {
	PRList   []interface{}
	TestFail int
	LastSeen *time.Time
	LogURLs  map[any] /* pr number/ -> log urls */ []string
}

// type TestFailEntryPriodic struct {
// 	PRList   []string
// 	TestFail int
// 	LastSeen *time.Time
// 	LogURLs  map[string] /* pr number -> log urls */ []string
// }

type TestFails struct {
	Score    int
	TestName string
	Fails    int
	LastSeen string
	PRList   []any
	Entry    TestFailEntry
}

// Result ...
type Result map[string]map[string][]Match
