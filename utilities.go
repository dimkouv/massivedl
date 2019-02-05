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
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	log.Fatal(err)
	return false
}

func getUserHomeDirectory() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
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

	reader := bufio.NewReader(in)

	fmt.Print("\n", msg)
	if defaultChoice {
		fmt.Print(" [Y/n]")
	} else {
		fmt.Print(" [y/N]")
	}
	fmt.Print(": ")

	text, _ := reader.ReadString('\n')

	if strings.HasSuffix(text, "\n") {
		text = strings.Split(text, "\n")[0]
	}

	reply := strings.ToLower(text)

	if strIndexOf(choicesTrue, reply) >= 0 {
		return true
	}

	if strIndexOf(choicesFalse, reply) >= 0 {
		return false
	}

	return defaultChoice
}
