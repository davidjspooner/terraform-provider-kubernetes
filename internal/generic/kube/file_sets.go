package kube

import (
	"bufio"
	"bytes"
	"fmt"
	htemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	ttemplate "text/template"
)

type ExpandedContent struct {
	Filename string
	LineNo   int
	Content  []byte
}

type ExpandedContentHandler interface {
	HandleExpandedContent(content *ExpandedContent) error
}

type ExpandedContentHandlerFunc func(content *ExpandedContent) error

func (f ExpandedContentHandlerFunc) HandleExpandedContent(content *ExpandedContent) error {
	return f(content)
}

type FileSetDef struct {
	GlobPaths     []string
	Variables     map[string]any
	TemplateType  string
	SplitYamlDocs bool
	//TODO SOPS decode
}

func (f FileSetDef) processDocument(ec *ExpandedContent, handler ExpandedContentHandler) error {
	w := &bytes.Buffer{}
	switch f.TemplateType {
	case "go/text":
		t := ttemplate.New("go-text")
		//todo funcs
		t, err := t.Parse(string(ec.Content))
		if err != nil {
			return err
		}
		t.Execute(w, f.Variables)
		ec.Content = w.Bytes()
	case "go/html":
		t := htemplate.New("go-text")
		//todo funcs
		t, err := t.Parse(string(ec.Content))
		if err != nil {
			return err
		}
		t.Execute(w, f.Variables)
		ec.Content = w.Bytes()
	default:
		if f.TemplateType != "" {
			return fmt.Errorf("unknown template type %s\nsupported=go/text,go/html", f.TemplateType)
		}
		if len(f.Variables) > 0 {
			return fmt.Errorf("variables are only supported when template_type is set")
		}
	}
	//TODO process the document
	err := handler.HandleExpandedContent(ec)
	if err != nil {
		return err
	}
	return nil
}

func (f FileSetDef) ExpandContent(handler ExpandedContentHandler) error {
	for _, globPath := range f.GlobPaths {
		realPaths, err := filepath.Glob(globPath)
		if err != nil {
			return err
		}
		if len(realPaths) == 0 {
			return fmt.Errorf("no files found for glob %s", globPath)
		}

		for _, realPath := range realPaths {
			stats, err := os.Stat(realPath)
			if err != nil {
				return err
			}
			if stats.IsDir() {
				return nil
			}
			file, err := os.Open(realPath)
			if err != nil {
				return err
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			ec := ExpandedContent{
				Filename: realPath,
				LineNo:   0,
				Content:  content,
			}
			fileExt := strings.ToLower(filepath.Ext(realPath))
			if (fileExt == ".yaml" || fileExt == ".yml") && f.SplitYamlDocs {
				buffer := &bytes.Buffer{}
				r := bytes.NewReader(content)
				scanner := bufio.NewScanner(r)

				// Scan the file line by line
				for scanner.Scan() {
					line := scanner.Text() // Get the current line as a string
					ec.LineNo++
					if strings.TrimRight(line, " \t") == "---" {
						// If we encounter a separator, handle the current content
						if buffer.Len() > 0 {
							ec.Content = buffer.Bytes()
							err = f.processDocument(&ec, handler)
							if err != nil {
								return err
							}
							buffer.Reset() // Clear the buffer for the next document
						}
					} else {
						buffer.WriteString(line + "\n") // Append the line to the buffer
					}
				}
				if buffer.Len() > 0 {
					// Handle the last document if there's any content left in the buffer
					ec.Content = buffer.Bytes()
					err = f.processDocument(&ec, handler)
					if err != nil {
						return err
					}
					buffer.Reset()
				}
				if err := scanner.Err(); err != nil {
					return err
				}
			} else {
				err = f.processDocument(&ec, handler)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type FileSetDefs []*FileSetDef

func (f FileSetDefs) ExpandContent(handler ExpandedContentHandler) error {
	for _, def := range f {
		err := def.ExpandContent(handler)
		if err != nil {
			return err
		}
	}
	return nil
}
