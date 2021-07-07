package filesystem

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

// Write creates a file with the content of the input string
func Write(filePath string, content string) bool {
	file, err := os.Create(filePath)

	if err != nil { // TO-DO handle the error
		panic(err)
	}

	defer file.Close() //Assures close will be called later

	_, err = file.WriteString(content)

	if err != nil { // TO-DO handle the error
		panic(err)
	}

	return true
}

// WriteTemp creates a new temporary file in a directory with the content of the input string and returns the file path
func WriteTemp(directoryPath string, content string) string {
	file, err := ioutil.TempFile(directoryPath, "temp-*.go")

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
	}

	_, err = file.WriteString(content)

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
	}

	err = file.Close() // Closing the file is required by Windows, if not the file gets locked

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
	}

	return file.Name()
}

// Remove deletes a file on a filePath
func Remove(filePath string) bool {
	err := os.Remove(filePath)

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
		return false
	}

	return true
}

// RemoveCache deletes a all the contents of the cache directory
func RemoveCache(cachePath string) bool {
	err := os.RemoveAll(cachePath)

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
		return false
	}

	err = os.MkdirAll(cachePath, 0666)

	if err != nil { // TO-DO handle the error
		log.Fatal(err)
		return false
	}

	return true
}

// WriteSerialized creates a file with the serialized JSON of an object
func WriteSerialized(object interface{}, filePath string) {
	objectJSON, err := json.Marshal(&object)

	if err != nil {
		//fmt.Println(err) //TO-DO
	}

	Write(filePath, string(objectJSON))
}
