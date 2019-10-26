package clitool

import (
	"io"
	"io/ioutil"
	"testing"
)

func TestAskUserBool(t *testing.T) {
	testCases := []struct {
		inputText     string
		defaultOption bool
		expectedRes   bool
	}{
		{"yes\n", false, true},
		{"no\n", false, false},
		{"\n", false, false},
		{"\n", true, true},
	}

	for _, testCase := range testCases {
		// simulates user input by passing a temp file
		in, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}

		_, err = io.WriteString(in, testCase.inputText)
		if err != nil {
			t.Fatal(err)
		}

		_, err = in.Seek(0, io.SeekStart)
		if err != nil {
			t.Fatal(err)
		}

		res := AskUserBool("automated input:", testCase.defaultOption, in)

		if res != testCase.expectedRes {
			t.Errorf("Test case with inputText=%#v, defaultOption=%v returned: %v",
				testCase.inputText, testCase.defaultOption, res)
		}

		if err = in.Close(); err != nil {
			t.Error(err)
		}
	}
}
