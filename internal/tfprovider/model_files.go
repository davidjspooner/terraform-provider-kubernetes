package tfprovider

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/kresource"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FilesModel struct {
	Paths        types.List   `tfsdk:"paths"`
	TemplateType types.String `tfsdk:"template_type"`
	Values       types.Map    `tfsdk:"values"`
}

func DefineFileListSchema(required bool) schema.Attribute {
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

func (fm *FilesModel) AddToStringMap(sm *kresource.StringMap) error {
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
			if err := sm.AddFileContents(match, templateType, values); err != nil {
				return fmt.Errorf("error adding file contents for path %q: %w", match, err)
			}
		}
	}
	return nil
}
