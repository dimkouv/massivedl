package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"
)

func writeToLog(res logEntry) {
	log.Println(res.url, res.name, res.result, res.nBytes, res.duration)
}

func fileOrPathExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func getUserHomeDirectory() (string, error) {
	usr, err := user.Current()
	return usr.HomeDir, err
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// index of value in slice of strings
func strIndexOf(s []string, v string) int {
	for i := 0; i < len(s); i++ {
		if strings.Compare(v, s[i]) == 0 {
			return i
		}
	}
	return -1
}

// ask user a yes/no question and get the result
func askUserBool(msg string, defaultChoice bool, in *os.File) bool {
	if in == nil {
		in = os.Stdin
	}

	choicesTrue := []string{"yes", "1", "y", "yeah"}
	choicesFalse := []string{"no", "0", "n", "nah"}

	if defaultChoice {
		fmt.Printf("\n %s [Y/n]: ", msg)
	} else {
		fmt.Printf("\n %s [y/N]: ", msg)
	}

	reader := bufio.NewReader(in)

	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	text = strings.ToLower(text)

	if strings.HasSuffix(text, "\n") {
		text = strings.Split(text, "\n")[0]
	}

	if strIndexOf(choicesTrue, text) >= 0 {
		return true
	}
	if strIndexOf(choicesFalse, text) >= 0 {
		return false
	}
	return defaultChoice
}

//Convert a float64 to time.Duration
//@param quantity - the quantity of time to convert
//@param unit - the time unit of conversion ("ns", "us" (or "µs"), "ms", "s", "m", "h")
func floatToDuration(tQuantity float64, tUnit string) time.Duration {
	stringTime := fmt.Sprintf("%g", tQuantity) + tUnit
	duration, err := time.ParseDuration(stringTime)

	if err != nil {
		log.Fatal(err)
	}

	return duration
}
