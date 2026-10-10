package parser

import (
	"encoding/json"
	"strings"
)

// Trailer is what a script reports about its response after its output
type Trailer struct {
	Status  int         `json:"status"`  // 0 when the script set none
	Headers [][2]string `json:"headers"` // name and value, in the order they were set
}

// SplitOutput separates a script's output from the trailer the runtime prints after the separator.
// The last separator wins, so a script printing the separator cannot spoof headers.
// The trailer is nil when the script exited before printing one (os.Exit, panic) or it is malformed.
func SplitOutput(stdout, separator string) (string, *Trailer) {
	i := strings.LastIndex(stdout, separator+" ")
	if i < 0 {
		return stdout, nil
	}
	var trailer Trailer
	if err := json.Unmarshal([]byte(stdout[i+len(separator)+1:]), &trailer); err != nil {
		return stdout, nil
	}
	return stdout[:i], &trailer
}
