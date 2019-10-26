package logging

import (
	"log"
	"time"
)

// LogEntry contains the fields that our log file will contain
type LogEntry struct {
	Url      string        // url of the file we tried to download
	Name     string        // name for the output file
	Result   bool          // whether or not the file was downloaded
	NBytes   uint64        // number of bytes of the downloaded file
	Duration time.Duration // how much time this download needed
}

// Print prints a LogEntry
func (l LogEntry) Print() {
	log.Println(l.Url, l.Name, l.Result, l.NBytes, l.Duration)
}
