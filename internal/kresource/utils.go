package kresource

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
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

func ParseSingleYamlManifest(content string) (unstructured.Unstructured, error) {
	yamlData := []byte(content)
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	obj := unstructured.Unstructured{}
	_, _, err := decoder.Decode(yamlData, nil, &obj)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return obj, nil
}
