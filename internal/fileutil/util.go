package fileutil

import (
	"os"
	"os/user"
)

// FileOrPathExists returns true/false whether or not the specified path exists
func FileOrPathExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

// GetUserHomeDirectory returns the full path of the user's home directory
func GetUserHomeDirectory() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	return usr.HomeDir, err
}
