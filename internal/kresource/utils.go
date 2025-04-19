package kresource

import (
	"fmt"
	"os"
	"strings"
)

func FirstNonNullString(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

var homeDir string

// TODO support windows ?
func ExpandEnv(path string) (string, error) {

	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE")
		}
		if homeDir == "" {
			return "", fmt.Errorf("HOME or USERPROFILE not set")
		}
	}

	betterPath := os.ExpandEnv(path)
	betterPath = strings.ReplaceAll(betterPath, "~", homeDir)
	if betterPath == "" {
		return "", fmt.Errorf("path %q is empty", path)
	}
	info, err := os.Stat(betterPath)
	if err == nil && info.IsDir() {
		betterPath += "/config"
		info, err = os.Stat(betterPath)
		if err == nil && info.IsDir() {
			return "", fmt.Errorf("path %q is a directory", betterPath)
		}
	}
	return betterPath, nil
}

//func ErrorIsNotFound(err error) bool {
//	if err == nil {
//		return false
//	}
//	errStr := err.Error()
//	for _, match := range []string{"not found", "no matches for kind", "not found:", "not found (get)"} {
//		if strings.Contains(errStr, match) {
//			return true
//		}
//	}
//	return false
//}
//
//func FirstNonEmpty[E constraints.Ordered](values ...E) E {
//	var null E
//	for _, v := range values {
//		if v != null {
//			return v
//		}
//
//	}
//	return values[len(values)-1]
//}
//
