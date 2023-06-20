package pkg

import (
	"encoding/base32"
	"io/ioutil"
	"os"
	"time"
)

// BlobStorage ...
type BlobStorage struct {
	path string
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

	files, err := ioutil.ReadDir(pathParam)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		info, err := os.Stat(pathParam + "/" + f.Name())

		if err != nil {
			return nil, err
		}

		modTime := info.ModTime()

		diff := time.Since(modTime)

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

func (s BlobStorage) Store(key string, value string) error {
	base64Key := base32.StdEncoding.EncodeToString([]byte(key))

	expectedPath := s.path + "/" + base64Key[:18]

	err := ioutil.WriteFile(expectedPath, []byte(value), 0755)

	return err

}

func (s BlobStorage) Retrieve(key string) (string, error) {
	base64Key := base32.StdEncoding.EncodeToString([]byte(key))

	expectedPath := s.path + "/" + base64Key[:18]

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		return "", nil
	}

	contents, err := ioutil.ReadFile(expectedPath)
	if err != nil {
		return "", err
	}

	return string(contents), nil

}
