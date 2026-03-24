package util

import (
	"os"
	"path/filepath"

	"github.com/mgenware/ku-builder/io2"
)

const kKuCacheDirName = ".ku-builder"

func WriteKuCacheFile(content string, paths []string) (string, error) {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(userDir, kKuCacheDirName)
	dirPath := filepath.Dir(path)

	io2.Mkdirp(dirPath)
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return path, nil
}
