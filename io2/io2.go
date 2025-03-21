package io2

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func pathExistsCore(path string) (os.FileInfo, error) {
	if fileInfo, err := os.Stat(path); err == nil {
		return fileInfo, nil
	} else if os.IsNotExist(err) {
		return nil, nil
	} else {
		return nil, err
	}
}

func FileExists(file string) bool {
	info, err := pathExistsCore(file)
	if err != nil {
		return false
	}
	return info != nil && !info.IsDir()
}

func DirectoryExists(dir string) bool {
	info, err := pathExistsCore(dir)
	if err != nil {
		return false
	}
	return info != nil && info.IsDir()
}

func FileMustExist(file string) string {
	if !FileExists(file) {
		panic("File does not exist: " + file)
	}
	return file
}

func DirectoryMustExist(dir string) string {
	if !DirectoryExists(dir) {
		panic("Directory does not exist: " + dir)
	}
	return dir
}

func ResolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}

func IsDirectoryEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func Mkdirp(dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
}

func CleanDir(dir string) {
	os.RemoveAll(dir)
	Mkdirp(dir)
}

func JoinCLIFlags(flags ...string) string {
	nonEmptyFlags := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag != "" {
			nonEmptyFlags = append(nonEmptyFlags, flag)
		}
	}
	return strings.Join(nonEmptyFlags, " ")
}

func PathMustExist(path string) string {
	if !FileExists(path) && !DirectoryExists(path) {
		panic("Path does not exist: " + path)
	}
	return path
}
