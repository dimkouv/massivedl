package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/dimkouv/massivedl/internal/logging"

	"github.com/dimkouv/massivedl/internal/clitool"

	"github.com/dimkouv/massivedl/internal/fileutil"
	"github.com/dimkouv/massivedl/internal/statistics"
	"github.com/dimkouv/massivedl/internal/timeutil"
)

// a dataEntry has the required information to download a file
// a dataEntry is normally loaded from a .csv file and is stored in a slice
type dataEntry struct {
	name string
	url  string
}

// cmdLineParams - Configuration struct
type cmdLineParams struct {
	ConcurrentRequests int     `json:"concurrentRequests"`
	EntriesFilepath    string  `json:"entriesFilepath"`
	SkippedLines       int     `json:"skippedLines"`
	OutputDir          string  `json:"outputDir"`
	MaxRetries         int     `json:"maxRetries"`
	Offset             int     `json:"offset"`
	DelayPerRequest    float64 `json:"delayPerRequest"`
}

// saveEntry - data required for saving/loading progress
type saveEntry struct {
	WorkingDirectory string                `json:"workingDirectory"`
	Parameters       cmdLineParams         `json:"cmdLineParams"`
	Stats            statistics.Statistics `json:"stats"`
}

var stats statistics.Statistics
var p cmdLineParams
var stopWorking bool // workers check this flag before taking a job

// loads data entries from a csv file.
// csv file entries be (output name, url)
// check examples/ for example .csv files
// @param filename - The full path of the .csv file to load
// @param offset - Number of lines to skip from the beginning
//                       of the csv file
func parseDownloadsFromCsv(filename string, offset int) []dataEntry {
	var entries []dataEntry

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Printf("error closing file: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(file)

	/* pass the skipped lines */
	for i := 0; i < offset; i++ {
		scanner.Scan()
	}
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ",", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, dataEntry{
			strings.Trim(parts[0], " "),
			strings.Trim(parts[1], " "),
		})
	}

	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return entries
}

func parseCmdLineParams() {
	var version = flag.Bool("v", false, "Print version info")
	var loadedFile = flag.String("l", "", "Saved progress file to load")
	var entriesFilepath = flag.String("i", "", "Input downloads csv file")
	var concurrentRequests = flag.Int("p", 10, "Number of parallel requests")
	var skippedLines = flag.Int("s", 1, "Number of skipped lines from input")
	var outputDir = flag.String("o", "downloads", "Directory to place downloads")
	var maxRetries = flag.Int("r", 3, "Number of retries for failed downloads")
	var delayPerRequest = flag.Float64("d", 1, "Delay per request in seconds")
	flag.Parse()

	if *version || *entriesFilepath == "" {
		PrintVersionInfo()
		os.Exit(0)
	}

	if *loadedFile != "" {
		p = loadProgress(*loadedFile)
	} else {
		p.EntriesFilepath = *entriesFilepath
		p.ConcurrentRequests = *concurrentRequests
		p.SkippedLines = *skippedLines
		p.OutputDir = *outputDir
		p.MaxRetries = *maxRetries
		p.DelayPerRequest = *delayPerRequest
	}
}

func getSaveFilesDirectory() string {
	homeDir, err := fileutil.GetUserHomeDirectory()
	if err != nil {
		log.Fatal(err)
	}

	saveFilesDirPath := path.Join(homeDir, ".massivedl")

	if !fileutil.FileOrPathExists(saveFilesDirPath) {
		if err = os.MkdirAll(saveFilesDirPath, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}

	return saveFilesDirPath
}

func getSaveFilePath() string {
	filename := fmt.Sprintf("%d_progress.save", timeutil.GetCurrentTimestamp())
	return path.Join(getSaveFilesDirectory(), filename)
}

func saveProgress() {
	var err error

	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var save saveEntry
	p.Offset = stats.TotalDownloads - stats.FilesRemaining - 1
	save.WorkingDirectory = workDir
	save.Parameters = p
	save.Stats = stats

	b, err := json.Marshal(save)
	if err != nil {
		log.Fatal(err)
	}

	saveFilePath := getSaveFilePath()

	err = ioutil.WriteFile(saveFilePath, b, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("\nProgress has been saved!")
		fmt.Println("Use the following command to continue downloading")
		fmt.Printf("\n\tmassivedl --load %s\n", saveFilePath)
	}
}

func loadProgress(saveFile string) cmdLineParams {
	var err error

	b, err := ioutil.ReadFile(saveFile)
	if err != nil {
		log.Fatal(err)
	}

	var l saveEntry
	err = json.Unmarshal(b, &l)
	if err != nil {
		log.Fatal(err)
	}

	// load statistics
	stats = l.Stats
	// reset stats that do not make sense to be loaded
	stats.AverageSpeedBytesPerSec = 0
	stats.AverageSpeedFilesPerSec = 0
	stats.SpeedBytesPerSec = 0
	stats.StartTime = time.Now()

	err = os.Chdir(l.WorkingDirectory)
	if err != nil {
		fmt.Println("(warning) The directory you executed massivedl in the past")
		fmt.Println("doesn't exist. If input file was a relative path then it might fail.")
	}

	return l.Parameters
}

func registerSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		<-sigChan
		stopWorking = true
		stats.Print()
		stats.PrintEnd()

		if clitool.AskUserBool("Do you want to save progress?", true, nil) {
			saveProgress()
		}

		os.Exit(0)
	}()
}

// Downloads a file on the specified url
// @param filepath - The file where the output will be saved
func download(url, filepath string, maxRetries int) logging.LogEntry {
	totalTries := 0
	logRow := logging.LogEntry{Url: url, Name: filepath, Result: false, NBytes: 0, Duration: 0}
	var response *http.Response
	var err error
	var file *os.File

	startTime := time.Now()
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()

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

		break
	}

	logRow.Duration = (time.Now()).Sub(startTime)

	// create subdirectories if they do not exist
	parts := strings.Split(filepath, "/")
	if len(parts) > 1 {
		if err = os.MkdirAll(strings.Join(parts[:len(parts)-1], "/"), os.ModePerm); err != nil {
			log.Fatalf("unable to create directories: %v", err)
		}
	}

	file, err = os.Create(filepath)
	if err != nil {
		log.Fatal(err)
		return logRow
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Printf("unable to close file: %v", err)
		}
	}()

	nBytes, err := io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
		return logRow
	}

	logRow.Result = true
	logRow.NBytes = uint64(nBytes)

	return logRow
}

func worker(_ int, jobs <-chan dataEntry, results chan<- logging.LogEntry) {
	for j := range jobs {
		if stopWorking {
			break
		}

		res := download(j.url, path.Join(p.OutputDir, j.name), p.MaxRetries)
		stats.Update(res)
		res.Print()
		results <- res

		time.Sleep(timeutil.FloatToDuration(p.DelayPerRequest, "s"))
	}
}

func run(_ cmdLineParams) {
	// flag that is raised on SIGINT signal
	stopWorking = false

	registerSignalHandlers()

	// create downloads dir if it doesn't exist
	if err := os.MkdirAll(p.OutputDir, os.ModePerm); err != nil {
		log.Fatalf("unable to create directories: %v", err)
	}

	// load urls - entries to download
	entries := parseDownloadsFromCsv(p.EntriesFilepath, p.SkippedLines+p.Offset)
	stats.TotalDownloads = len(entries)

	// set number of workers from command line parameters
	numWorkers := p.ConcurrentRequests

	// create log file
	f, err := os.OpenFile(path.Join(getSaveFilesDirectory(), "massivedl.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			fmt.Printf("unable to close file: %v", err)
		}
	}()

	// redirect logger output on the log file
	log.SetOutput(f)

	// create jobs channel
	jobs := make(chan dataEntry, stats.TotalDownloads)

	// create results channel
	results := make(chan logging.LogEntry, stats.TotalDownloads)

	// print output header
	stats.PrintHeader()

	// run output goroutine
	// this goroutine updates the statics in stdout
	go func() {
		for !stopWorking {
			stats.Print()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// init worker goroutines
	for i := 0; i < numWorkers; i++ {
		go worker(i, jobs, results)
	}

	// start sending jobs
	for i := 0; i < stats.TotalDownloads; i++ {
		jobs <- entries[i]
	}
	close(jobs)

	// catch results
	for i := 0; i < stats.TotalDownloads; i++ {
		<-results
	}

	// print the final statistics
	stats.Print()
	stats.PrintEnd()
}

func main() {
	// initialize statistics
	// statistics should be initialized before parsing cmdLineParams
	// parsing command line params might alter the statistics when loading progress
	stats = statistics.New()

	// parse command line parameters
	parseCmdLineParams()

	// start downloading
	run(p)
}
