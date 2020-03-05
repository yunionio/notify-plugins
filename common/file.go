// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

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
