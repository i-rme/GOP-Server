package cache

import (
	"pfg/src/repository/filesystem"
	"pfg/src/server/logs"
	"time"
)

type entry struct {
	value     string
	timestamp time.Time
}

var (
	//Relation defines the data structure that will bound the file path of the source file to the executable
	Relation map[string]entry
)

func init() {

	logs.WriteDebug("INFO: Cache system initializing")
	Relation = make(map[string]entry)

}

//Put inserts a new entry in the cache map
func Put(key, value, name string) {

	key = filesystem.Clean(key)                                       // Avoid duplicated cache entries by cleaning the path
	value += "/b001/exe/" + filesystem.FilenameWithoutExtension(name) // We need to append the path to the executable

	Relation[key] = entry{
		value:     value,
		timestamp: time.Now(),
	}
}

//Get reads a entry from the cache map
func Get(key string) (string, time.Time) {
	return Relation[key].value, Relation[key].timestamp
}

//GetIfFresh reads a entry from the cache map if its fresh
func GetIfFresh(sourceFile string) string {
	sourceFile = filesystem.Clean(sourceFile) // Avoid duplicated cache entries by cleaning the path
	cacheFile, cacheModTime := Get(sourceFile)

	fileModTime := filesystem.ModificationTime(sourceFile)

	if cacheModTime.After(fileModTime) { // If cache is fresh return the cache file path
		return cacheFile
	}

	return "" // Return empty string if not

}
