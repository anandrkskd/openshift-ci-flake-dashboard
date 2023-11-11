package pkg

import (
	"time"
)

// Match ...
type Match struct {
	FileType  string   `json:"filename"`
	Context   []string `json:"context,omitempty"`
	MoreLines int      `json:"moreLines,omitempty"`
}

// TestFailEntry ...
type TestFailEntry struct {
	PRList   []int
	TestFail int
	LastSeen *time.Time
	LogURLs  map[int] /* pr number -> log urls */ []string
}

type TestFailEntryPriodic struct {
	PRList   []string
	TestFail int
	LastSeen *time.Time
	LogURLs  map[string] /* pr number -> log urls */ []string
}

type periodicJobData struct {
	failure        string
	clusterVersion string
	url            string
	flag           bool
}

// Result ...
type Result map[string]map[string][]Match

// BlobStorage ...
type BlobStorage struct {
	path string
}

type Config struct {
	Pull      bool   `json:"pull"`
	Periodic  bool   `json:"periodic"`
	Regex     string `json:"regex"`
	RepoName  string `json:"repoName"`
	RepoOrg   string `json:"repoOrg"`
	SearchStr string `json:"searchStr"`
}
