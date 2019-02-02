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

type logEntry struct {
	url    string
	name   string
	result bool
}

type dataEntry struct {
	name string
	url  string
}

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
		entries = append(entries, dataEntry{parts[0], parts[1]})
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return entries
}

func download(url, name string) logEntry {
	logRow := logEntry{url, name, false}

	response, e := http.Get(url)
	if e != nil {
		return logRow
	}
	defer response.Body.Close()

	file, err := os.Create(name)
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
	b := 8
	filename := "data.csv"
	ignoredLines := 1
	outputDir := "downloads"
	totalDownloaded := 0

	f, err := os.OpenFile("massivedl.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	entries := readData(filename, ignoredLines)
	n := len(entries)

	os.MkdirAll(outputDir, os.ModePerm)

	for i := 0; i < n; {
		if i+b >= n {
			b = n - i
		}

		var wg sync.WaitGroup
		logEntries := make(chan logEntry)
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

		/* logs for this batch */
		go func() {
			for logI := range logEntries {
				if logI.result == true {
					totalDownloaded++
				}
				log.Println(logI.name, logI.url, logI.result)
			}
		}()

		wg.Wait()

		i += b
	}

	fmt.Println("\n**COMPLETED**")
	fmt.Println("Total downloads:", totalDownloaded)
	fmt.Println("Total failures :", n-totalDownloaded)
	fmt.Println("Check massivedl.log for more information")
}
