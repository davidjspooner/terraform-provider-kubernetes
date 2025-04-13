package pmodel

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	htmp_template "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	text_template "text/template"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Files struct {
	Paths        types.List   `tfsdk:"paths"`
	TemplateType types.String `tfsdk:"template_type"`
	Values       types.Map    `tfsdk:"values"`
}

func FileListSchema(required bool) schema.Attribute {
	result := schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"paths": schema.ListAttribute{
				MarkdownDescription: "List of paths to files",
				ElementType:         types.StringType,
				Required:            true,
			},
			"values": schema.MapAttribute{
				MarkdownDescription: "Map of values to be used in the file. Requires template_type to be set",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"template_type": schema.StringAttribute{
				MarkdownDescription: "Type of template to be used (text or html)",
				Optional:            true,
			},
		},
	}
	if required {
		result.Required = true
	} else {
		result.Optional = true
	}
	return result
}

type StringMap struct {
	base64Encoded bool
	data          map[string]string
}

func (sm *StringMap) SetBase64Encoded(b64 bool) {
	if sm.base64Encoded != b64 {
		if b64 {
			for k, v := range sm.data {
				b := []byte(v)
				sm.data[k] = base64.StdEncoding.EncodeToString(b)
			}
		} else {
			for k, v := range sm.data {
				b, err := base64.StdEncoding.DecodeString(v)
				if err != nil {
					continue // Ignore errors for now
				}
				sm.data[k] = string(b)
			}
		}
	}
	sm.base64Encoded = b64
}

func (sm *StringMap) Add(key, value string) error {
	if key == "" {
		return errors.New("key in data set is empty")
	}
	_, ok := sm.data[key]
	if ok {
		return fmt.Errorf("key %q already exists in data set", key)
	}
	if sm.data == nil {
		sm.data = make(map[string]string)
	}
	if sm.base64Encoded {
		b := []byte(value)
		value = base64.StdEncoding.EncodeToString(b)
	}
	sm.data[key] = value
	return nil
}

func (sm *StringMap) AddBase64(key, value string) error {
	if key == "" {
		return errors.New("key in data set is empty")
	}
	_, ok := sm.data[key]
	if ok {
		return fmt.Errorf("key %q already exists in data set", key)
	}
	if sm.data == nil {
		sm.data = make(map[string]string)
	}
	if !sm.base64Encoded {
		b, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return fmt.Errorf("error decoding base64 value for key %q: %w", key, err)
		}
		value = string(b)
	}
	sm.data[key] = value
	return nil
}

func (sm *StringMap) Get(key string) (string, error) {
	if sm == nil || len(sm.data) == 0 {
		return "", fmt.Errorf("key %q not found in empty data set", key)
	}
	value, ok := sm.data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in data set", key)
	}
	if sm.base64Encoded {
		b, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", fmt.Errorf("error decoding base64 value for key %q: %w", key, err)
		}
		value = string(b)
	}
	return value, nil
}

func (sm *StringMap) GetBase64(key string) (string, error) {
	if sm == nil || len(sm.data) == 0 {
		return "", fmt.Errorf("key %q not found in empty data set", key)
	}
	value, ok := sm.data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in data set", key)
	}
	if !sm.base64Encoded {
		b := []byte(value)
		value = base64.StdEncoding.EncodeToString(b)
	}
	return value, nil
}

func (sm *StringMap) AddMaps(data, binaryData types.Map) error {
	var elements map[string]string

	diags := data.ElementsAs(context.Background(), &elements, false)
	if diags.HasError() {
		return fmt.Errorf("error getting data map")
	}

	for k, v := range elements {
		if k == "" {
			return errors.New("key is empty string")
		}
		if err := sm.Add(k, v); err != nil {
			return err
		}
	}
	diags = binaryData.ElementsAs(context.Background(), &elements, false)
	if diags.HasError() {
		return fmt.Errorf("error getting data map")
	}
	for k, v := range elements {
		if k == "" {
			return errors.New("key is empty string")
		}
		if err := sm.AddBase64(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (sm *StringMap) addFileContents(path string, TemplateType string, values map[string]string) error {
	if path == "" {
		return errors.New("path in data set is empty")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %q: %w", path, err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading file %q: %w", path, err)
	}
	basename := filepath.Base(path)
	switch TemplateType {
	case "text":
		t := text_template.New(path)
		t, err = t.Parse(string(b))
		if err != nil {
			return fmt.Errorf("error parsing text template %q: %w", path, err)
		}
		var expanded bytes.Buffer
		err = t.Execute(&expanded, values)
		if err != nil {
			return fmt.Errorf("error executing text template %q: %w", path, err)
		}
		sm.Add(basename, expanded.String())
	case "html":
		t := htmp_template.New(path)
		t, err = t.Parse(string(b))
		if err != nil {
			return fmt.Errorf("error parsing html template %q: %w", path, err)
		}
		var expanded bytes.Buffer
		err = t.Execute(&expanded, values)
		if err != nil {
			return fmt.Errorf("error executing html template %q: %w", path, err)
		}
		sm.Add(basename, expanded.String())
	case "":
		if len(values) > 0 {
			return fmt.Errorf("values provided but template_type is not set")
		}
		sm.Add(basename, string(b))
	default:
		return fmt.Errorf("unsupported template type %q", TemplateType)
	}

	return nil
}

func (sm *StringMap) AddFileModel(fm *Files) error {
	if fm == nil {
		return nil
	}
	templateType := fm.TemplateType.ValueString()
	values := make(map[string]string)

	diags := fm.Values.ElementsAs(context.Background(), &values, false)
	if diags.HasError() {
		return fmt.Errorf("error getting values map")
	}

	var paths []string
	ctx := context.Background()
	diags = fm.Paths.ElementsAs(ctx, &paths, false)
	if diags.HasError() {
		return fmt.Errorf("error getting paths")
	}
	var err error
	for _, path := range paths {
		if path == "" {
			return errors.New("path is empty")
		}
		// Treat the path value as a glob if it contains a wildcard
		var matches []string
		if strings.ContainsAny(path, "*?[]") {
			matches, err = filepath.Glob(path)
			if err != nil {
				return fmt.Errorf("error processing glob pattern %q: %w", path, err)
			}
			if matches == nil {
				return fmt.Errorf("no files match the glob pattern %q", path)
			}
		} else {
			matches = []string{path}
		}
		for _, match := range matches {
			if err := sm.addFileContents(match, templateType, values); err != nil {
				return fmt.Errorf("error adding file contents for path %q: %w", match, err)
			}
		}
	}
	return nil
}

func (sm *StringMap) GetUnstructured() map[string]string {
	if sm == nil || len(sm.data) == 0 {
		return nil
	}
	copy := make(map[string]string, len(sm.data))
	for k, v := range sm.data {
		copy[k] = v
	}
	return copy
}

func (sm *StringMap) ForEach(ctx context.Context, fn func(key, value string) error) error {
	if sm == nil || len(sm.data) == 0 {
		return nil
	}
	for k, v := range sm.data {
		if err := fn(k, v); err != nil {
			return err
		}
	}
	return nil
}
