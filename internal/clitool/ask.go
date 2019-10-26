package clitool

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dimkouv/massivedl/internal/sliceutil"
)

// AskUserBool asks a true/false question with text defined in msg and returns the result that was read from in.
// If the user does not specify any option the the defaultChoice value is returned.
func AskUserBool(msg string, defaultChoice bool, in *os.File) bool {
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

	if sliceutil.StrIndexOf(choicesTrue, text) >= 0 {
		return true
	}

	if sliceutil.StrIndexOf(choicesFalse, text) >= 0 {
		return false
	}

	return defaultChoice
}
