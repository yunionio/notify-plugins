package utils

import "os"

func CheckDir(dir string) error {
	if IsExist(dir) {
		return nil
	}
	err := os.Mkdir(dir, 0770)
	if err != nil {
		return err
	}
	return nil
}

func IsExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
