package main

import (
	"fmt"
	"strings"
)

// PrintVersionInfo prints a description of this application
func PrintVersionInfo() {
	usage := [...]string{
		"NAME",
		"\tmassivedl v" + Version + " - Download a list of files in parallel",
		"\nSYNOPSIS",
		"\tmassivedl [OPTION]...",
		"\nDESCRIPTION",
		"\tmassivedl is a free utility for non-interactive download of files from the web.",
		"\tThis utility can be used to download a large list of files from the web in parallel batches.",
		"\tYou can get really good results when the server you're downloading from has low response time.",
		"\nEXAMPLE",
		"\tmassivedl -p 10 -i data.csv -s 1 -o downloads -d 2.3",
		"\nAUTHOR",
		"\tdimkouv <dimkouv@protonmail.com>",
		"\tContributions at: https://github.com/dimkouv/massivedl",
		"\nBUILD INFO",
		"\tVersion:    " + Version,
		"\tBuildstamp: " + Buildstamp,
		"\tGithash:    " + Githash,
	}
	fmt.Println(strings.Join(usage[:], "\n"))
}
