package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// a logEntry has the information a log entry needs
type logEntry struct {
	url      string        // url of the file we tried to download
	name     string        // name for the output file
	result   bool          // whether or not the file was downloaded
	nBytes   uint64        // number of bytes of the downloaded file
	duration time.Duration // how much time this download needed
}

// a dataEntry has the required information to download a file
// a dataEntry is normally loaded from a .csv file and is stored in a slice
type dataEntry struct {
	name string
	url  string
}

type cmdLineParams struct {
	concurrentRequests int
	entriesFilepath    string
	skippedLines       int
	outputDir          string
	maxRetries         int
}

type statistics struct {
	totalDownloaded         int
	totalFailed             int
	totalDownloadedBytes    uint64
	averageSpeedFilesPerSec float64
	speedBytesPerSec        float64
	startTime               time.Time
	filesRemaining          int
	averageSpeedBytesPerSec float64
}

var stats statistics
var p cmdLineParams
var n int // total downloads

// loads data entries from a csv file.
// csv file entries be (output name, url)
// check examples/ for example .csv files
// @param filename - The full path of the .csv file to load
// @param skippedLines - Number of lines to skip from the beginning
//                       of the csv file
func parseDownloadsFromCsv(filename string, skippedLines int) []dataEntry {
	var entries []dataEntry

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	/* pass the skipped lines */
	for i := 0; i < skippedLines; i++ {
		scanner.Scan()
	}

	/* read the lines */
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, dataEntry{
			strings.Trim(parts[0], " "),
			strings.Trim(parts[1], " "),
		})
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return entries
}

// Downloads a file on the specified url
// @param filepath - The file where the output will be saved
func download(url, filepath string, maxRetries int) logEntry {
	totalTries := 0
	logRow := logEntry{url, filepath, false, 0, 0}
	var response *http.Response
	var err error
	var file *os.File

	startTime := time.Now()

	for {
		if totalTries > maxRetries {
			return logRow
		}

		response, err = http.Get(url)
		if err != nil {
			log.Println("[RETRY]", totalTries, url, filepath)
			totalTries++
			continue
		}
		defer response.Body.Close()
		break
	}

	logRow.duration = (time.Now()).Sub(startTime)

	file, err = os.Create(filepath)
	if err != nil {
		log.Fatal(err)
		return logRow
	}
	defer file.Close()

	nBytes, err := io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
		return logRow
	}

	logRow.result = true
	logRow.nBytes = uint64(nBytes)

	return logRow
}

func printUsage() {
	usage := [...]string{
		"NAME",
		"\tmassivedl - Download a list of files in parallel batches",
		"\nSYNOPSIS",
		"\tmassivedl [OPTION]...",
		"\nDESCRIPTION",
		"\tmassivedl is a free utility for non-interactive download of files from the web.",
		"\tThis utility can be used to download a large list of files from the web in parallel batches.",
		"\tYou can get really good results when the server you're downloading from has low response time.",
		"\nOPTIONS",
		"\t-p <int> (default=10)          ::: Maximum number of parallel requests",
		"\t-i <str>                       ::: Input csv file with the list of urls",
		"\t-s <int> (default=0)           ::: Number of skipped lines from input csv",
		"\t-o <str> (default='downloads') ::: Directory to place the downloads",
		"\t-r <int> (default=1)           ::: Maximum number of retries for failed downloads",
		"\nEXAMPLE",
		"\tmassivedl -p 10 -i data.csv -s 1 -o downloads",
		"\nAUTHOR",
		"\tdimkouv <dimkouv@protonmail.com>",
		"\tContributions at: https://github.com/dimkouv/massivedl",
		"\n",
	}
	fmt.Println(strings.Join(usage[:], "\n"))
}

func parseCmdLineParams() cmdLineParams {
	p := cmdLineParams{10, "", 0, "downloads", 1}
	var err error

	for i := 0; i < len(os.Args)-1; i++ {
		if strings.Compare(os.Args[i], "-p") == 0 {
			// -p ::: number of parallel requests pool
			p.concurrentRequests, err = strconv.Atoi(os.Args[i+1])

			if err != nil {
				printUsage()
				log.Fatal("Error parsing command line parameters")
			}
		} else if strings.Compare(os.Args[i], "-i") == 0 {
			// -i ::: entries file path
			p.entriesFilepath = os.Args[i+1]
		} else if strings.Compare(os.Args[i], "-s") == 0 {
			// -s ::: number of skipped lines
			p.skippedLines, err = strconv.Atoi(os.Args[i+1])

			if err != nil {
				printUsage()
				log.Fatal("Error parsing command line parameters")
			}
		} else if strings.Compare(os.Args[i], "-o") == 0 {
			// -o ::: output - downloads directory
			p.outputDir = os.Args[i+1]
		} else if strings.Compare(os.Args[i], "-r") == 0 {
			// -r ::: maximum number of retries
			p.maxRetries, err = strconv.Atoi(os.Args[i+1])

			if err != nil || p.maxRetries < 0 {
				printUsage()
				log.Fatal("Error parsing command line parameters")
			}
		}
	}

	if strings.Compare(p.entriesFilepath, "") == 0 {
		printUsage()
		log.Fatal("You have to provide input csv file using -i cmd line param.")
	}

	return p
}

func updateStatistics(log logEntry, statsMutex *sync.Mutex) {
	statsMutex.Lock()

	durationSoFar := (time.Now()).Sub(stats.startTime)

	if log.result == true {
		stats.totalDownloaded++
	} else {
		stats.totalFailed++
	}

	stats.totalDownloadedBytes += log.nBytes
	stats.speedBytesPerSec = float64(log.nBytes) / log.duration.Seconds()
	stats.averageSpeedFilesPerSec = float64(stats.totalDownloaded) / durationSoFar.Seconds()
	stats.averageSpeedBytesPerSec = float64(stats.totalDownloadedBytes) / (durationSoFar.Seconds())
	stats.filesRemaining = n - (stats.totalDownloaded + stats.totalFailed)

	statsMutex.Unlock()
}

func worker(id int, jobs <-chan dataEntry, results chan<- logEntry, statsMutex *sync.Mutex) {
	for j := range jobs {
		res := download(j.url, path.Join(p.outputDir, j.name), p.maxRetries)
		updateStatistics(res, statsMutex)
		results <- res
	}
}

func printStatistics() {
	fmt.Printf("\r%-9d | %-10d | %-10.2f | %-11.2f | %-7.2f | %-10d | %-11.2f |",
		stats.totalDownloaded,
		stats.totalFailed,
		float64(stats.totalDownloadedBytes)/1000000.0,
		stats.averageSpeedFilesPerSec, stats.speedBytesPerSec/1000000.0,
		stats.filesRemaining,
		stats.averageSpeedBytesPerSec/1000000,
	)
}

func printStatsHeader() {
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

func printStatsEnd() {
	durationSoFar := (time.Now()).Sub(stats.startTime)

	fmt.Println("\n\nTotal time:", durationSoFar)
	fmt.Println("Thanks for using massivedl.")
}

func main() {
	// parse command line parameters
	p = parseCmdLineParams()

	// initialize statistics
	stats = statistics{}
	stats.startTime = time.Now()
	printStatsHeader()

	// statsMutex for locking statistics
	var statsMutex = &sync.Mutex{}

	// load urls - entries to download
	entries := parseDownloadsFromCsv(p.entriesFilepath, p.skippedLines)
	n = len(entries)

	// create downloads dir if it doesn't exist
	os.MkdirAll(p.outputDir, os.ModePerm)

	// set number of workers from command line parameters
	numWorkers := p.concurrentRequests

	// create log file
	f, err := os.OpenFile("massivedl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	// redirect logger output on the log file
	log.SetOutput(f)

	// create jobs channel
	jobs := make(chan dataEntry, n)

	// create results channel
	results := make(chan logEntry, n)

	// run output coroutine
	// this coroutine updates the statics in stdout
	go func() {
		for {
			printStatistics()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// init worker coroutines
	for i := 0; i < numWorkers; i++ {
		go worker(i, jobs, results, statsMutex)
	}

	// start sending jobs
	for i := 0; i < n; i++ {
		jobs <- entries[i]
	}
	close(jobs)

	// catch results
	for i := 0; i < n; i++ {
		<-results
	}

	printStatistics()
	printStatsEnd()
}
