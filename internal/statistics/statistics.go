package statistics

import (
	"fmt"
	"sync"
	"time"

	"github.com/dimkouv/massivedl/internal/logging"
)

// Statistics - statistics about the downloads
type Statistics struct {
	lock                    *sync.RWMutex
	TotalDownloads          int       `json:"totalDownloads"`
	TotalDownloaded         int       `json:"totalDownloaded"`
	TotalFailed             int       `json:"totalFailed"`
	TotalDownloadedBytes    uint64    `json:"totalDownloadedBytes"`
	AverageSpeedFilesPerSec float64   `json:"averageSpeedFilesPerSec"`
	SpeedBytesPerSec        float64   `json:"speedBytesPerSec"`
	StartTime               time.Time `json:"startTime"`
	FilesRemaining          int       `json:"filesRemaining"`
	AverageSpeedBytesPerSec float64   `json:"averageSpeedBytesPerSec"`
}

// New returns a new Statistics instance with start time the current time
func New() Statistics {
	return Statistics{StartTime: time.Now(), lock: &sync.RWMutex{}}
}

// Update updates the statistics from a new log entry
func (stats *Statistics) Update(log logging.LogEntry) {
	stats.lock.Lock()
	defer stats.lock.Unlock()

	durationSoFar := (time.Now()).Sub(stats.StartTime)

	if log.Result {
		stats.TotalDownloaded++
	} else {
		stats.TotalFailed++
	}

	stats.TotalDownloadedBytes += log.NBytes
	stats.SpeedBytesPerSec = float64(log.NBytes) / log.Duration.Seconds()
	stats.AverageSpeedFilesPerSec = float64(stats.TotalDownloaded) / durationSoFar.Seconds()
	stats.AverageSpeedBytesPerSec = float64(stats.TotalDownloadedBytes) / (durationSoFar.Seconds())
	stats.FilesRemaining = stats.TotalDownloads - (stats.TotalDownloaded + stats.TotalFailed)

}

// PrintHeader prints the header of the statistics
func (stats *Statistics) PrintHeader() {
	fmt.Printf("\n%-9s | %-10s | %-10s | %-11s | %-7s | %-10s | %-11s |\n",
		"Downloads",
		"Failures",
		"Total mB",
		"Files/Sec",
		"mB/Sec",
		"Remaining",
		"Avg mB/Sec",
	)
}

// Print prints a row with the statistics
func (stats *Statistics) Print() {
	stats.lock.RLock()
	defer stats.lock.RUnlock()

	fmt.Printf("\r%-9d | %-10d | %-10.2f | %-11.2f | %-7.2f | %-10d | %-11.2f |",
		stats.TotalDownloaded,
		stats.TotalFailed,
		float64(stats.TotalDownloadedBytes)/1000000.0,
		stats.AverageSpeedFilesPerSec, stats.SpeedBytesPerSec/1000000.0,
		stats.FilesRemaining,
		stats.AverageSpeedBytesPerSec/1000000,
	)
}

// PrintEnd is called on program exit and prints some useful final stats
func (stats *Statistics) PrintEnd() {
	stats.lock.RLock()
	defer stats.lock.RUnlock()

	durationSoFar := (time.Now()).Sub(stats.StartTime)

	fmt.Println("\n\nTotal time:", durationSoFar)
	fmt.Println("Thank you for using massivedl")
}
