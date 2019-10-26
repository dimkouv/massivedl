package sliceutil

import "testing"

func TestStrIndexOf(t *testing.T) {
	testCases := []struct {
		arr         []string
		target      string
		expectedPos int
	}{
		{
			[]string{"hello", "my", "name", "is", "1"},
			"1",
			4,
		},
		{
			[]string{"hello", "my", "name", "is", "1"},
			"name",
			2,
		},
		{
			[]string{"hello", "my", "name", "is", "1"},
			"la la",
			-1,
		},
	}

	for _, testCase := range testCases {
		foundPos := StrIndexOf(testCase.arr, testCase.target)
		if foundPos != testCase.expectedPos {
			t.Errorf("target=%s array=%v expectedPos=%d received foundPos=%d",
				testCase.target, testCase.arr, testCase.expectedPos, foundPos)
		}
	}
}
