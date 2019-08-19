package utils

import (
	"os"
	"path/filepath"

	"yunion.io/x/pkg/errors"
)

func CheckDir(dir string, subDirs ...string) error {
	if IsExist(dir) {
		return nil
	}
	err := os.Mkdir(dir, 0770)
	if err != nil {
		return errors.Wrapf(err, "Create dir %s failed", dir)
	}
	for _, subDir := range subDirs {
		dirPath := filepath.Join(dir, subDir)
		err := os.Mkdir(dirPath, 0770)
		if err != nil {
			return errors.Wrapf(err, "Create dir %s failed", dirPath)
		}
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
