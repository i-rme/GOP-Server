package filesystem

import (
	"io/ioutil"
	"os"
	"time"
)

// Exists checks if a file exists on the file system
func Exists(filePath string) bool {

	/*	PERFORMANCE TO-DO
		Exists accounts for 10% of time for cached requests
	*/

	info, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		return false
	}

	if info.IsDir() {
		return false
	}

	return true
}

// DirectoryExists checks if a directory exists on the file system
func DirectoryExists(filePath string) bool {

	/*	PERFORMANCE TO-DO
		Exists accounts for 10% of time for cached requests
	*/

	info, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

// Read returns the contents of a file
func Read(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)

	return string(content), err
}

// ModificationTime returns the last modification date from a file path
func ModificationTime(filePath string) time.Time {
	file, err := os.Stat(filePath)

	if err != nil { // TO-DO handle the error
		panic(err)
	}

	return file.ModTime()
}

// IsCacheFresh returns true if the cache was modified after the original file
func IsCacheFresh(filePath string, cachePath string) bool {
	if !Exists(cachePath) {
		return false
	}

	fileModTime := ModificationTime(filePath)
	cacheModTime := ModificationTime(cachePath)

	return cacheModTime.After(fileModTime)
}
