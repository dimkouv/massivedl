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
)

// a logEntry has the information a log entry needs
type logEntry struct {
	url    string // url of the file we tried to download
	name   string // name for the output file
	result bool   // whether or not the file was downloaded
}

// a dataEntry has the required information to download a file
// a dataEntry is normally loaded from a .csv file and is stored in a slice
type dataEntry struct {
	name string
	url  string
}

type cmdLineParams struct {
	batchSize       int
	entriesFilepath string
	skippedLines    int
	outputDir       string
	maxRetries      int
}

// loads data entries from a csv file.
// csv file entries be (output name, url)
// check examples/ for example .csv files
// @param filename - The full path of the .csv file to load
// @param skippedLines - Number of lines to skip from the beginning
//                       of the csv file
func readData(filename string, skippedLines int) []dataEntry {
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
	logRow := logEntry{url, filepath, false}
	var response *http.Response
	var err error
	var file *os.File

	for {
		if totalTries > maxRetries {
			return logRow
		}

		response, err = http.Get(url)
		if err != nil {
			fmt.Println("Trying again...", url)
			totalTries++
			continue
		}
		defer response.Body.Close()
		break
	}

	file, err = os.Create(filepath)
	if err != nil {
		fmt.Println(err)
		return logRow
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Println(err)
		return logRow
	}

	logRow.result = true
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
		"\t-b <int> ::: Size of a batch ::: default 10",
		"\t-i <str> ::: Input csv file with the list of urls",
		"\t-s <int> ::: Number of skipped lines from input csv ::: default 0",
		"\t-o <str> ::: Directory to place the downloads ::: default 'downloads'",
		"\t-r <int> ::: Maximum number of retries for failed downloads ::: default 1",
		"\nEXAMPLE",
		"\tmassivedl -b 10 -i data.csv -s 1 -o downloads",
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
		if strings.Compare(os.Args[i], "-b") == 0 {
			// -b ::: size of a batch
			p.batchSize, err = strconv.Atoi(os.Args[i+1])

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

func main() {
	p := parseCmdLineParams()

	totalDownloaded := 0

	// mutex for locking stats
	var mutex = &sync.Mutex{}

	// load urls - entries to download
	entries := readData(p.entriesFilepath, p.skippedLines)
	n := len(entries)

	// create downloads dir if it doesn't exist
	os.MkdirAll(p.outputDir, os.ModePerm)

	b := p.batchSize

	fmt.Printf("massivedl about to download %d files\n", n)

	// create log file
	f, err := os.OpenFile("massivedl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	// redirect logger output on the log file
	log.SetOutput(f)

	var wgGlbl sync.WaitGroup
	wgGlbl.Add(n)

	for i := 0; i < n; {
		// fix batch size for the last iteration
		if i+b >= n {
			b = n - i
		}

		// create a channel for fetching the result logs for this batch
		logEntries := make(chan logEntry)

		// create a workgroup (synchronization var) for current batch
		var wg sync.WaitGroup
		wg.Add(b)

		fmt.Printf("Current batch: %d (~%d %%)\n", i/b, 100*(i+b-1)/n)
		fmt.Printf("Range:[%d, %d]\n\n", i, i+b-1)

		/* call download function for this batch */
		for j := 0; j < b; j++ {
			go func(idx int) {
				logEntries <- download(entries[idx].url, path.Join(p.outputDir, entries[idx].name), p.maxRetries)
				wg.Done()
			}(i + j)
		}

		/* write logs for this batch */
		go func() {
			for logI := range logEntries {
				wgGlbl.Done()
				if logI.result == true {
					mutex.Lock()
					totalDownloaded++
					mutex.Unlock()
				}
				fmt.Println(logI.name, logI.url, logI.result)
			}
		}()

		// wait for this batch
		wg.Wait()
		i += b
	}

	wgGlbl.Wait() // wait for all logs to be written
	fmt.Println("\n**COMPLETED**")
	fmt.Println("Total downloads:", totalDownloaded)
	fmt.Println("Total failures :", n-totalDownloaded)
	fmt.Println("Check massivedl.log for more information")
}
