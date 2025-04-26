package kube

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ParsedDocument struct {
	Manifest string
	Parsed   unstructured.Unstructured
	Source   struct {
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
	Manifest string
	Source   struct {
		Filename string
		Line     int
	}
}

func (doc *unparsedDocument) ExpandTemplateAndParse(variables map[string]any) (*ParsedDocument, error) {
	// Implement the parsing logic here
	// This is a placeholder implementation

	s := doc.Manifest

	if len(variables) > 0 {
		//create a template from the manifest and expand it with the variables
		source := fmt.Sprintf("%s [line:%d]", doc.Source.Filename, doc.Source.Line)
		t, err := template.New(source).Parse(s)
		if err != nil {
			return nil, err
		}
		var buf strings.Builder
		err = t.Execute(&buf, variables)
		if err != nil {
			return nil, err
		}
		s = buf.String()
	}

	unstructuredObj, err := ParseSingleYamlManifest(s)
	if err != nil {
		return nil, err
	}
	parsedDoc := &ParsedDocument{
		Manifest: s,
		Parsed:   unstructuredObj,
		Source: struct {
			File string
			Line int
		}{
			File: doc.Source.Filename,
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
		return nil, fmt.Errorf("missing apiVersion in manifest: %s line %d ", doc.Source.Filename, doc.Source.Line)
	}
	if parsedDoc.Kind == "" {
		return nil, fmt.Errorf("missing kind in manifest: %s line %d ", doc.Source.Filename, doc.Source.Line)
	}
	if parsedDoc.Metadata.Name == "" {
		return nil, fmt.Errorf("missing name in manifest: %s line %d ", doc.Source.Filename, doc.Source.Line)
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
					Manifest: s,
					Source: struct {
						Filename string
						Line     int
					}{
						Filename: filePath,
						Line:     lineNumber,
					},
				})
				currentManifest.Reset()
			}
			continue
		} else {
			// Add the line to the current manifest
			currentManifest.WriteString(line + "\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	s := strings.TrimSpace(currentManifest.String())
	if len(s) > 0 {
		documents = append(documents, unparsedDocument{
			Manifest: s,
			Source: struct {
				Filename string
				Line     int
			}{
				Filename: filePath,
				Line:     lineNumber,
			},
		})
		currentManifest.Reset()
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents found in file: %s", filePath)
	}

	return documents, nil
}

func SplitAndExpandTemplates(templateFiles []string, variables map[string]any) ([]*ParsedDocument, error) {
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
			parsedDoc, err := unparsedDoc.ExpandTemplateAndParse(variables)
			if err != nil {
				return nil, err
			}
			parsedDocuments = append(parsedDocuments, parsedDoc)
		}
	}

	return parsedDocuments, nil
}
