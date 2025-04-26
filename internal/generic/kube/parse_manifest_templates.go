package kube

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ParsedDocument struct {
	FullManifest unstructured.Unstructured
	Source       struct {
		File string
		Line int
	}
	APIVersion string
	Kind       string
	Metadata   struct {
		Name      string
		Namespace string
	}
}

type unparsedDocument struct {
	FullManifest string
	Source       struct {
		File string
		Line int
	}
}

func (doc *unparsedDocument) Parse() (*ParsedDocument, error) {
	// Implement the parsing logic here
	// This is a placeholder implementation

	unstructuredObj, err := ParseSingleYamlManifest(doc.FullManifest)
	if err != nil {
		return nil, err
	}
	parsedDoc := &ParsedDocument{
		FullManifest: unstructuredObj,
		Source: struct {
			File string
			Line int
		}{
			File: doc.Source.File,
			Line: doc.Source.Line,
		},
		APIVersion: unstructuredObj.GetAPIVersion(),
		Kind:       unstructuredObj.GetKind(),
		Metadata: struct {
			Name      string
			Namespace string
		}{
			Name:      unstructuredObj.GetName(),
			Namespace: unstructuredObj.GetNamespace(),
		},
	}
	if parsedDoc.APIVersion == "" {
		return nil, fmt.Errorf("missing apiVersion in manifest: %s line %d ", doc.Source.File, doc.Source.Line)
	}
	if parsedDoc.Kind == "" {
		return nil, fmt.Errorf("missing kind in manifest: %s line %d ", doc.Source.File, doc.Source.Line)
	}
	if parsedDoc.Metadata.Name == "" {
		return nil, fmt.Errorf("missing name in manifest: %s line %d ", doc.Source.File, doc.Source.Line)
	}

	return parsedDoc, nil
}

func readDocumentsFromFileAndSplit(filePath string) ([]unparsedDocument, error) {
	// Implement the file reading and splitting logic here
	// This is a placeholder implementation
	var documents []unparsedDocument
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Read the yaml file and split it into documents with line numbers. Split on lines that are "---"
	scanner := bufio.NewScanner(f)
	currentManifest := strings.Builder{}
	lineNumber := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		if line == "---" {
			s := strings.TrimSpace(currentManifest.String())
			if len(s) > 0 {
				documents = append(documents, unparsedDocument{
					FullManifest: s,
					Source: struct {
						File string
						Line int
					}{
						File: filePath,
						Line: lineNumber,
					},
				})
				currentManifest.Reset()
			}
			continue
		} else {
			// Add the line to the current manifest
			currentManifest.WriteString(line + "\n")
		}

		currentManifest.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return documents, nil
}

func ParseManifestTemplates(templateFiles []string, variables map[string]any) ([]*ParsedDocument, error) {
	// Implement the parsing logic here
	// This is a placeholder implementation
	var parsedDocuments []*ParsedDocument

	var globbedFiles []string
	for _, templateFile := range templateFiles {
		globbed, err := filepath.Glob(templateFile)
		if err != nil {
			return nil, err
		}
		globbedFiles = append(globbedFiles, globbed...)
	}

	for _, filePath := range globbedFiles {
		unparsedDocuments, err := readDocumentsFromFileAndSplit(filePath)
		if err != nil {
			return nil, err
		}
		for _, unparsedDoc := range unparsedDocuments {
			parsedDoc, err := unparsedDoc.Parse()
			if err != nil {
				return nil, err
			}
			parsedDocuments = append(parsedDocuments, parsedDoc)
		}
	}

	return parsedDocuments, nil
}
