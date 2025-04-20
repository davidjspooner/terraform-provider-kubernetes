package kresource

import (
	"encoding/base64"
	"fmt"
	html_template "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	text_template "text/template"
)

type StringMap struct {
	textData          map[string]string
	base64EncodedData map[string]string
}

func (sm *StringMap) Clear() {
	sm.textData = nil
	sm.base64EncodedData = nil
}

func (sm *StringMap) validateUniqueKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	for _, badChar := range "/\\?%*:|\"<>.\n\r\t\b\f" {
		if strings.ContainsRune(key, badChar) {
			return fmt.Errorf("key %q contains %q", key, badChar)
		}
	}
	lowerKey := strings.ToLower(key)
	for exisingKey := range sm.textData {
		if lowerKey == strings.ToLower(exisingKey) {
			return fmt.Errorf("key %q already exists", key)
		}
	}
	for exisingKey := range sm.base64EncodedData {
		if lowerKey == strings.ToLower(exisingKey) {
			return fmt.Errorf("key %q already exists", key)
		}
	}
	return nil
}

func (sm *StringMap) AddText(key, value string) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.textData == nil {
		sm.textData = make(map[string]string)
	}
	sm.textData[key] = value
	return nil
}
func (sm *StringMap) AddBase64(key, value string) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.base64EncodedData == nil {
		sm.base64EncodedData = make(map[string]string)
	}
	sm.base64EncodedData[key] = value
	return nil
}
func (sm *StringMap) EncodeAsBase64AndAdd(key, value string) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.base64EncodedData == nil {
		sm.base64EncodedData = make(map[string]string)
	}
	base64Value := base64.StdEncoding.EncodeToString([]byte(value))
	sm.base64EncodedData[key] = base64Value
	return nil
}

func (sm *StringMap) AddTextFileContents(path, templateType string, values map[string]string) error {
	if sm.textData == nil {
		sm.textData = make(map[string]string)
	}
	basename := filepath.Base(path)
	err := sm.validateUniqueKey(basename)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %q: %w", path, err)
	}

	if values == nil {
		sm.textData[basename] = string(content)
		return nil
	}

	switch templateType {
	case "text":
		// No processing needed, just read the file
		t := text_template.New(basename)
		t, err = t.Parse(string(content))
		if err != nil {
			return fmt.Errorf("error parsing template %q: %w", basename, err)
		}
		var sb strings.Builder
		err = t.Execute(&sb, values)
		if err != nil {
			return fmt.Errorf("error executing template %q: %w", basename, err)
		}
		sm.textData[basename] = sb.String()
	case "html":
		t := html_template.New(basename)
		t, err = t.Parse(string(content))
		if err != nil {
			return fmt.Errorf("error parsing template %q: %w", basename, err)
		}
		var sb strings.Builder
		err = t.Execute(&sb, values)
		if err != nil {
			return fmt.Errorf("error executing template %q: %w", basename, err)
		}
		sm.textData[basename] = sb.String()
	default:
		return fmt.Errorf("unknown template type %q", templateType)
	}
	return nil
}

func (sm *StringMap) AddFileContentAsBase64(path string) error {
	if sm.base64EncodedData == nil {
		sm.base64EncodedData = make(map[string]string)
	}
	basename := filepath.Base(path)
	err := sm.validateUniqueKey(basename)
	if err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %q: %w", path, err)
	}
	defer f.Close()
	sb := strings.Builder{}
	e := base64.NewEncoder(base64.StdEncoding, &sb)
	_, err = io.Copy(e, f)
	if err != nil {
		return fmt.Errorf("error encoding file %q: %w", path, err)
	}
	e.Close()
	sm.base64EncodedData[basename] = sb.String()
	return nil
}

func (sm *StringMap) GetUnstructuredText() map[string]interface{} {
	if sm.textData == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range sm.textData {
		result[k] = v
	}
	return result
}
func (sm *StringMap) GetUnstructuredBase64() map[string]interface{} {
	if sm.base64EncodedData == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range sm.base64EncodedData {
		result[k] = v
	}
	return result
}
func (sm *StringMap) ForEachTextContent(f func(key string, value string) error) error {
	if sm.textData == nil {
		return nil
	}
	for k, v := range sm.textData {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (sm *StringMap) ForEachBase64Content(f func(key string, value string) error) error {
	if sm.base64EncodedData == nil {
		return nil
	}
	for k, v := range sm.base64EncodedData {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}
