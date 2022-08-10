package filesystem

import (
	"os"
)

func CreateDirectory(dirName string) bool {
	src, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirName, 0755)
		if errDir != nil {
			panic(err)
		}
		return true
	}

	if src.Mode().IsRegular() {
		return false
	}

	return false
}

func Exist(pathname string) bool {
	_, err := os.Stat(pathname)
	return os.IsNotExist(err) == false
}
