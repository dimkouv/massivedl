package main

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestStrIndexOf(t *testing.T) {
	s := []string{"hello", "my", "name", "is", "1"}

	if strIndexOf(s, "1") != 4 {
		t.Errorf("Invalid index of '1")
	}

	if strIndexOf(s, "name") != 2 {
		t.Errorf("Invalid index of 'name'")
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

	_, err = in.Seek(0, os.SEEK_SET)
	if err != nil {
		t.Fatal(err)
	}

	res := askUserBool("enter:", false, in)

	if res != true {
		t.Errorf("Wrong response, should return true.")
	}
}
