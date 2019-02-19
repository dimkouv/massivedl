package main

import (
	"io"
	"io/ioutil"
	"testing"
)

func TestStrIndexOf(t *testing.T) {
	s := []string{"hello", "my", "name", "is", "1"}

	if strIndexOf(s, "1") != 4 {
		t.Errorf("Invalid index of '1\n")
	}

	if strIndexOf(s, "name") != 2 {
		t.Errorf("Invalid index of 'name'\n")
	}
}

func TestAskUserBool(t *testing.T) {
	// simulates user input by passing a temp file

	in, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	_, err = io.WriteString(in, "yes\n")
	if err != nil {
		t.Fatal(err)
	}

	_, err = in.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	res := askUserBool("enter:", false, in)

	if res != true {
		t.Errorf("Wrong response, should return true.\n")
	}
}

func TestParseDownloadsFromCsv(t *testing.T) {
	downloads := parseDownloadsFromCsv("./examples/list-of-photos.csv", 1)

	if len(downloads) != 12 {
		t.Errorf("examples/list-of-photos.csv returned %d entries instead of 12\n", len(downloads))
	}
}
