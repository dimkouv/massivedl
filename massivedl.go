package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

// CmdLineParams - Configuration struct
type CmdLineParams struct {
	ConcurrentRequests int     `json:"concurrentRequests"`
	EntriesFilepath    string  `json:"entriesFilepath"`
	SkippedLines       int     `json:"skippedLines"`
	OutputDir          string  `json:"outputDir"`
	MaxRetries         int     `json:"maxRetries"`
	Offset             int     `json:"offset"`
	DelayPerRequest    float64 `json:"delayPerRequest"`
}

// Statistics - statistics about the downloads
type Statistics struct {
	TotalDownloaded         int       `json:"totalDownloaded"`
	TotalFailed             int       `json:"totalFailed"`
	TotalDownloadedBytes    uint64    `json:"totalDownloadedBytes"`
	AverageSpeedFilesPerSec float64   `json:"averageSpeedFilesPerSec"`
	SpeedBytesPerSec        float64   `json:"speedBytesPerSec"`
	StartTime               time.Time `json:"startTime"`
	FilesRemaining          int       `json:"filesRemaining"`
	AverageSpeedBytesPerSec float64   `json:"averageSpeedBytesPerSec"`
}

// SaveEntry - data required for saving/loading progress
type SaveEntry struct {
	WorkingDirectory string        `json:"workingDirectory"`
	Parameters       CmdLineParams `json:"cmdLineParams"`
	Stats            Statistics    `json:"stats"`
}

var stats Statistics
var p CmdLineParams
var n int            // total downloads
var stopWorking bool // workers check this flag before tkaing a job

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
	defer file.Close()

	scanner := bufio.NewScanner(file)

	/* pass the skipped lines */
	for i := 0; i < offset; i++ {
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

func parseCmdLineParams() CmdLineParams {
	p := CmdLineParams{10, "", 0, "downloads", 1, 0, 0.0}
	var err error

	if strIndexOf(os.Args, "--help") >= 0 {
		// just print help message and exit
		printUsageAndExit()
	} else if len(os.Args) == 3 && strings.Compare("--load", os.Args[1]) == 0 {
		// check if user wants to load a saved progress
		p = loadProgress(os.Args[2])
	} else {
		// read parameters
		for i := 0; i < len(os.Args)-1; i++ {
			if strings.Compare(os.Args[i], "-p") == 0 {
				// -p ::: number of parallel requests pool
				p.ConcurrentRequests, err = strconv.Atoi(os.Args[i+1])

				if err != nil {
					printUsage()
					log.Fatal("Error parsing command line parameters")
				}
			} else if strings.Compare(os.Args[i], "-i") == 0 {
				// -i ::: entries file path
				p.EntriesFilepath = os.Args[i+1]
			} else if strings.Compare(os.Args[i], "-s") == 0 {
				// -s ::: number of skipped lines
				p.SkippedLines, err = strconv.Atoi(os.Args[i+1])

				if err != nil {
					printUsage()
					log.Fatal("Error parsing command line parameters")
				}
			} else if strings.Compare(os.Args[i], "-o") == 0 {
				// -o ::: output - downloads directory
				p.OutputDir = os.Args[i+1]
			} else if strings.Compare(os.Args[i], "-r") == 0 {
				// -r ::: maximum number of retries
				p.MaxRetries, err = strconv.Atoi(os.Args[i+1])

				if err != nil || p.MaxRetries < 0 {
					printUsage()
					log.Fatal("Error parsing command line parameters")
				}
			} else if strings.Compare(os.Args[i], "-d") == 0 {
				// -d ::: delay in seconds between each unparallel request starting time
				p.DelayPerRequest, err = strconv.ParseFloat(os.Args[i+1], 64)

				if err != nil || p.DelayPerRequest < 0 {
					printUsage()
					log.Fatal("Error parsing command line parameters")
				}

			}
		}
	}

	if strings.Compare(p.EntriesFilepath, "") == 0 {
		fmt.Println(
			"You have to provide input csv file using -i cmd line param.\n",
			"\rUse: massivedl --help for full reference.",
		)
		os.Exit(1)
	}

	return p
}

func printUsage() {
	usage := [...]string{
		"NAME",
		"\tmassivedl v" + Version + " - Download a list of files in parallel batches",
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
		"\t-d <float64> (default=0.0)     ::: Delay (in seconds) between each unparallel request",
		"\nEXAMPLE",
		"\tmassivedl -p 10 -i data.csv -s 1 -o downloads",
		"\nAUTHOR",
		"\tdimkouv <dimkouv@protonmail.com>",
		"\tContributions at: https://github.com/dimkouv/massivedl",
		"\nBUILD INFO",
		"\tVersion:    " + Version,
		"\tBuildstamp: " + Buildstamp,
		"\tGithash:    " + Githash,
	}
	fmt.Println(strings.Join(usage[:], "\n"))
}

func printUsageAndExit() {
	printUsage()
	os.Exit(0)
}

func updateStatistics(log logEntry, statsMutex *sync.Mutex) {
	statsMutex.Lock()

	durationSoFar := (time.Now()).Sub(stats.StartTime)

	if log.result == true {
		stats.TotalDownloaded++
	} else {
		stats.TotalFailed++
	}

	stats.TotalDownloadedBytes += log.nBytes
	stats.SpeedBytesPerSec = float64(log.nBytes) / log.duration.Seconds()
	stats.AverageSpeedFilesPerSec = float64(stats.TotalDownloaded) / durationSoFar.Seconds()
	stats.AverageSpeedBytesPerSec = float64(stats.TotalDownloadedBytes) / (durationSoFar.Seconds())
	stats.FilesRemaining = n - (stats.TotalDownloaded + stats.TotalFailed)

	statsMutex.Unlock()
}

func printStatistics() {
	fmt.Printf("\r%-9d | %-10d | %-10.2f | %-11.2f | %-7.2f | %-10d | %-11.2f |",
		stats.TotalDownloaded,
		stats.TotalFailed,
		float64(stats.TotalDownloadedBytes)/1000000.0,
		stats.AverageSpeedFilesPerSec, stats.SpeedBytesPerSec/1000000.0,
		stats.FilesRemaining,
		stats.AverageSpeedBytesPerSec/1000000,
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
	durationSoFar := (time.Now()).Sub(stats.StartTime)

	fmt.Println("\n\nTotal time:", durationSoFar)
	fmt.Println("Thanks for using massivedl.")
}

func getSaveFilesDirectory() string {
	homeDir := getUserHomeDirectory()
	saveFilesDirPath := path.Join(homeDir, ".massivedl")

	if !fileOrPathExists(saveFilesDirPath) {
		os.MkdirAll(saveFilesDirPath, os.ModePerm)
	}

	return saveFilesDirPath
}

func getSaveFilePath() string {
	filename := fmt.Sprintf("%d_progress.save", getCurrentTimestamp())
	return path.Join(getSaveFilesDirectory(), filename)
}

// saves current progress to a save file
func saveProgress() {
	var err error

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var save SaveEntry
	p.Offset = n - stats.FilesRemaining - 1
	save.WorkingDirectory = wd
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

// loads progress from a saved progress file
func loadProgress(saveFile string) CmdLineParams {
	var err error

	b, err := ioutil.ReadFile(saveFile)
	if err != nil {
		log.Fatal(err)
	}

	var l SaveEntry
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
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)

	go func() {
		<-sigc
		stopWorking = true
		printStatistics()
		printStatsEnd()

		if askUserBool("Do you want to save progress?", true, nil) == true {
			saveProgress()
		}

		os.Exit(0)
	}()
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

	// create subdirs if they do not exist
	parts := strings.Split(filepath, "/")
	if len(parts) > 1 {
		path := strings.Join(parts[:len(parts)-1], "/")
		os.MkdirAll(path, os.ModePerm)
	}

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

func worker(id int, jobs <-chan dataEntry, results chan<- logEntry, statsMutex *sync.Mutex) {
	for j := range jobs {
		if stopWorking {
			break
		}

		res := download(j.url, path.Join(p.OutputDir, j.name), p.MaxRetries)
		updateStatistics(res, statsMutex)
		writeToLog(res)
		results <- res
	}
}

func run(params CmdLineParams, stats Statistics) {
	// flag that is raised on SIGINT signal
	stopWorking = false

	// statsMutex for locking statistics
	var statsMutex = &sync.Mutex{}

	// register signal handlers
	registerSignalHandlers()

	// create downloads dir if it doesn't exist
	os.MkdirAll(p.OutputDir, os.ModePerm)

	// load urls - entries to download
	entries := parseDownloadsFromCsv(p.EntriesFilepath, p.SkippedLines+p.Offset)
	n = len(entries)

	// set number of workers from command line parameters
	numWorkers := p.ConcurrentRequests

	// create log file
	f, err := os.OpenFile(path.Join(getSaveFilesDirectory(), "massivedl.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
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
		for !stopWorking {
			printStatistics()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// init worker coroutines
	for i := 0; i < numWorkers; i++ {
		go worker(i, jobs, results, statsMutex)
	}

	// print output header
	printStatsHeader()
	// start sending jobs
	for i := 0; i < n; i++ {
		jobs <- entries[i]
	}
	close(jobs)

	// catch results
	for i := 0; i < n; i++ {
		<-results
	}

	// print the final statistics
	printStatistics()
	printStatsEnd()
}

func main() {
	// initialize statistics
	// statistics should be initialized before parsing cmdLineParams
	// parsing command line params might alter the statistics when loading progress
	stats = Statistics{}
	stats.StartTime = time.Now()

	// parse command line parameters
	p = parseCmdLineParams()

	// start downloading
	run(p, stats)
}
