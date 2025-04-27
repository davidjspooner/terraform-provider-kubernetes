package tfparts

import (
	"fmt"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type FileSetModel struct {
	Paths        types.List   `tfsdk:"paths"`
	TemplateType types.String `tfsdk:"template_type"`
	Variables    types.Map    `tfsdk:"variables"`
}

type FileSetModelList struct {
	FileSets []FileSetModel `tfsdk:"file_sets"`
}

func (f *FileSetModelList) GetFileSetDefs() kube.FileSetDefs {
	if f == nil {
		return nil
	}
	var fileSets kube.FileSetDefs
	for _, fileSet := range f.FileSets {

		tfGlob := fileSet.Paths.Elements()
		globPaths := make([]string, len(tfGlob))
		for i, path := range tfGlob {
			if path.IsNull() || path.IsUnknown() {
				continue
			}
			globPaths[i] = path.(types.String).ValueString()
		}
		variables := make(map[string]any)
		tfVars := fileSet.Variables.Elements()
		for i, v := range tfVars {
			if v.IsNull() || v.IsUnknown() {
				continue
			}
			variables[i] = v.(types.String).ValueString()
		}

		fileSetDef := &kube.FileSetDef{
			GlobPaths:    globPaths,
			TemplateType: fileSet.TemplateType.ValueString(),
			Variables:    variables,
		}
		fileSets = append(fileSets, fileSetDef)
	}
	return fileSets
}

func FileSetsResourceAttributes(required bool) map[string]rschema.Attribute {
	result := map[string]rschema.Attribute{
		"file_sets": rschema.ListNestedAttribute{
			MarkdownDescription: "List of file sets to process",
			NestedObject: rschema.NestedAttributeObject{
				Attributes: map[string]rschema.Attribute{
					"paths": schema.ListAttribute{
						MarkdownDescription: "List of paths to files",
						ElementType:         types.StringType,
						Required:            true,
					},
					"variables": rschema.MapAttribute{
						MarkdownDescription: "Map of variables to be used in template expansions. Requires template_type to be set",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"template_type": rschema.StringAttribute{
						MarkdownDescription: "Type of template to be used (text or html)",
						Optional:            true,
					},
				},
			},
			Required: required,
			Optional: !required,
		},
	}
	return result
}

func FileSetsDatasourceAttributes(required bool) map[string]dschema.Attribute {
	result := map[string]dschema.Attribute{
		"file_sets": dschema.ListNestedAttribute{
			MarkdownDescription: "List of file sets to process",
			NestedObject: dschema.NestedAttributeObject{
				Attributes: map[string]dschema.Attribute{
					"paths": dschema.ListAttribute{
						MarkdownDescription: "List of paths to files",
						ElementType:         types.StringType,
						Required:            true,
					},
					"variables": dschema.MapAttribute{
						MarkdownDescription: "Map of variables to be used in template expansions. Requires template_type to be set",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"template_type": dschema.StringAttribute{
						MarkdownDescription: "Type of template to be used (text or html)",
						Optional:            true,
					},
				},
			},
			Required: required,
			Optional: !required,
		},
	}
	return result
}

var DocumentElementAttrType = map[string]attr.Type{
	"text": types.StringType,
	"source": types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"file": types.StringType,
			"line": types.Int64Type,
		},
	},
	"api_version": types.StringType,
	"kind":        types.StringType,
	"metadata": types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":      types.StringType,
			"namespace": types.StringType,
		},
	},
}

func GenerateDocumentList(fsds kube.FileSetDefs) (types.Map, diag.Diagnostics) {
	parsedDocsMap := make(map[string]attr.Value)

	var result types.Map
	var diags diag.Diagnostics

	var handler kube.ExpandedContentHandlerFunc = func(ec *kube.ExpandedContent) error {
		content := string(ec.Content)
		u, err := kube.ParseSingleYamlManifest(content)
		if err != nil {
			return fmt.Errorf("error parsing manifest: %s", err)
		}

		kind := u.GetKind()
		if kind == "" {
			return fmt.Errorf("error parsing manifest: kind is empty")
		}
		name := u.GetName()
		if name == "" {
			return fmt.Errorf("error parsing manifest: name is empty")
		}

		namespace := u.GetNamespace()

		value := map[string]attr.Value{
			"api_version": types.StringValue(u.GetAPIVersion()),
			"kind":        types.StringValue(kind),
			"text":        types.StringValue(content),
		}
		value["source"], diags = basetypes.NewObjectValue(
			map[string]attr.Type{
				"file": types.StringType,
				"line": types.Int64Type,
			},
			map[string]attr.Value{
				"file": types.StringValue(ec.Filename),
				"line": types.Int64Value(int64(ec.LineNo)),
			},
		)
		value["metadata"], diags = basetypes.NewObjectValue(
			map[string]attr.Type{
				"name":      types.StringType,
				"namespace": types.StringType,
			},
			map[string]attr.Value{
				"name":      types.StringValue(name),
				"namespace": types.StringValue(namespace),
			},
		)
		object, diags := basetypes.NewObjectValue(DocumentElementAttrType, value)
		if diags.HasError() {
			return DiagsToGoError(diags)
		}
		key := fmt.Sprintf("%s:%s:%s", kind, namespace, name)
		_, exists := parsedDocsMap[key]
		if exists {
			return fmt.Errorf("duplicate manifest found for kind=%q namespace=%q name=%q at %s [%d]", kind, namespace, name, ec.Filename, ec.LineNo)
		}
		parsedDocsMap[key] = object
		return nil
	}

	err := fsds.ExpandContent(handler)
	if err != nil {
		diags.AddError("Error expanding content", err.Error())
		return result, diags
	}

	if len(parsedDocsMap) == 0 {
		details := "No documents found matching any of the provided file paths"
		diags.AddError("No documents found", details)
		return result, diags
	}

	result, diags = basetypes.NewMapValue(
		types.ObjectType{
			AttrTypes: DocumentElementAttrType,
		},
		parsedDocsMap,
	)
	return result, diags
}
