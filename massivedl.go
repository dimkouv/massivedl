package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
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

// loads data entries from a csv file.
// csv file entries be (output name, url)
// check examples/ for example .csv files
// @param filename - The full path of the .csv file to load
// @param ignoredLines - Number of lines to skip from the beginning
//                       of the csv file
func readData(filename string, ignoredLines int) []dataEntry {
	var entries []dataEntry

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	/* pass the ignored lines */
	for i := 0; i < ignoredLines; i++ {
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
func download(url, filepath string) logEntry {
	logRow := logEntry{url, filepath, false}

	response, e := http.Get(url)
	if e != nil {
		return logRow
	}
	defer response.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		fmt.Println(err)
		return logRow
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return logRow
	}

	time.Sleep(1 * time.Second)
	logRow.result = true
	return logRow
}

func main() {
	// @TODO replace with cmd line params
	b := 8                        // number of downloads per batch
	entriesFilepath := "data.csv" // path of input urls
	ignoredLines := 1             // skipped lines of input csv file
	outputDir := "downloads"      // output directory

	totalDownloaded := 0

	// create log file
	f, err := os.OpenFile("massivedl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	// redirect logger output on the log file
	log.SetOutput(f)

	// load urls - entries to download
	entries := readData(entriesFilepath, ignoredLines)
	n := len(entries)

	// create downloads dir if it doesn't exist
	os.MkdirAll(outputDir, os.ModePerm)

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
				logEntries <- download(entries[i].url, path.Join(outputDir, entries[idx].name))
				wg.Done()
				return
			}(i + j)
		}

		/* write logs for this batch */
		go func() {
			for logI := range logEntries {
				if logI.result == true {
					totalDownloaded++
				}
				log.Println(logI.name, logI.url, logI.result)
			}
		}()

		// wait for this batch
		wg.Wait()
		i += b
	}

	fmt.Println("\n**COMPLETED**")
	fmt.Println("Total downloads:", totalDownloaded)
	fmt.Println("Total failures :", n-totalDownloaded)
	fmt.Println("Check massivedl.log for more information")
}
