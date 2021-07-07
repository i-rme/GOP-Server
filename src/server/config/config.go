package config

import (
	"encoding/json"
	"fmt"
	"pfg/src/repository/filesystem"
	"time"
)

var (
	//Port specifies the port number that the server will listen to
	Port int

	// Address specifies the address for the server to listen on
	Address string

	//ReadTimeout is the time from when the connection is accepted to when the request body is fully read
	ReadTimeout time.Duration

	// WriteTimeout is the time from the end of the request header read to the end of the response write
	WriteTimeout time.Duration

	// IdleTimeout is the Keep-Alive timeout
	IdleTimeout time.Duration

	// MaxHeaderBytes is the maximum permitted size of the headers in an HTTP request
	MaxHeaderBytes int

	// DocumentRoot is the directory where the go, gop and static files are placed
	DocumentRoot string

	// CacheRoot is directory where the cache is stored
	CacheRoot string

	// LogRoot is directory where logs are stored
	LogRoot string

	// UploadsRoot is directory where uploaded files are stored
	UploadsRoot string

	// Aliases defines the relation between urls to be mapped and the script executed
	Aliases map[string]string

	// DenyFrom defines a list of CIDR IP ranges to block from our server
	DenyFrom []string

	// ServerSignature is a string with the server name and version
	ServerSignature string

	// DirectoryListingScript defines the path to the GOP script that handles Directory Listing
	DirectoryListingScript string

	// IPDeniedScript defines the path to the GOP script that handles IP Denied error
	IPDeniedScript string

	// SessionCookieName defines the name of the Session Cookie
	SessionCookieName string

	// GopHeadersSeparator is a string used internally to divide the GOP output and the headers
	GopHeadersSeparator string

	// LoggingLevel defines the detail of the logs: debug, basic, errors
	LoggingLevel string

	// GopKeySalt defines a string that will be used as salt for generating internal keys like the one used in file upload
	GopKeySalt string

	// RateLimitEnabled defines if the Rate Limiting feature is enabled or not
	RateLimitEnabled bool

	// RateLimitRate defines the number of requests per unit of RateLimitPeriod that will trigger the Rate Limiting
	RateLimitRate int

	// RateLimitPeriod defines the number of seconds used in the Rate Limiting calculation
	RateLimitPeriod int

	// RateLimitedScript defines the path to the GOP script that handles IP Rate Limited error
	RateLimitedScript string

	// MySQLSupportEnabled defines if MySQL database support is enabled or not
	MySQLSupportEnabled bool

	// RunScriptsAsNobody defines if GOP scripts should run without privileges
	RunScriptsAsNobody bool
)

//Configuration defines the data type of the configuration file
type Configuration struct {
	Port                   int
	Address                string
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	MaxHeaderBytes         int
	DocumentRoot           string
	CacheRoot              string
	LogRoot                string
	UploadsRoot            string
	Aliases                map[string]string
	DenyFrom               []string
	ServerSignature        string
	DirectoryListingScript string
	IPDeniedScript         string
	SessionCookieName      string
	GopHeadersSeparator    string
	LoggingLevel           string
	GopKeySalt             string
	RateLimitEnabled       bool
	RateLimitRate          int
	RateLimitPeriod        int
	RateLimitedScript      string
	MySQLSupportEnabled    bool
	RunScriptsAsNobody     bool
}

func init() {

	configFile := ReadSerialized("config/configuration.json")

	Port = configFile.Port
	Address = configFile.Address
	ReadTimeout = configFile.ReadTimeout
	WriteTimeout = configFile.WriteTimeout
	IdleTimeout = configFile.IdleTimeout
	MaxHeaderBytes = configFile.MaxHeaderBytes
	DocumentRoot = configFile.DocumentRoot
	CacheRoot = configFile.CacheRoot
	LogRoot = configFile.LogRoot
	UploadsRoot = configFile.UploadsRoot
	Aliases = configFile.Aliases
	DenyFrom = configFile.DenyFrom
	ServerSignature = configFile.ServerSignature
	DirectoryListingScript = configFile.DirectoryListingScript
	IPDeniedScript = configFile.IPDeniedScript
	SessionCookieName = configFile.SessionCookieName
	GopHeadersSeparator = configFile.GopHeadersSeparator
	LoggingLevel = configFile.LoggingLevel
	GopKeySalt = configFile.GopKeySalt
	RateLimitEnabled = configFile.RateLimitEnabled
	RateLimitRate = configFile.RateLimitRate
	RateLimitPeriod = configFile.RateLimitPeriod
	RateLimitedScript = configFile.RateLimitedScript
	MySQLSupportEnabled = configFile.MySQLSupportEnabled
	RunScriptsAsNobody = configFile.RunScriptsAsNobody
}

// ReadSerialized reads a config file with serialized JSON and returns an object
func ReadSerialized(filePath string) Configuration {

	var object Configuration

	objectJSON, err := filesystem.Read(filePath)

	if err != nil {
		fmt.Println("ERROR: ReadSerialized was unable to read the file. (Config error)")
		panic(err)
	}

	objectJSONString := []byte(objectJSON)    //JSON string to byte array
	json.Unmarshal(objectJSONString, &object) //JSON to Configuration

	_, err = json.Marshal(&object)

	if err != nil {
		//logs.Println(err) //TO-DO
	}

	return object
}
