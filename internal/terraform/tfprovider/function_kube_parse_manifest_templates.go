package tfprovider

import (
	"context"
	"math/big"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// FunctionKubeParseTemplateFiles implements the terraform function.Function interface.
type FunctionKubeParseTemplateFiles struct {
}

// NewFunctionKubeParseTemplateFiles creates a new instance of FunctionKubeParseTemplateFiles.
func NewFunctionKubeParseTemplateFiles() function.Function {
	return &FunctionKubeParseTemplateFiles{}
}

// Metadata provides metadata for the function.
func (f *FunctionKubeParseTemplateFiles) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "kube_parse_manifest_files"
}

func (f *FunctionKubeParseTemplateFiles) Definition(ctx context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Parameters: []function.Parameter{
			function.ListParameter{
				Name:        "template_files",
				Description: "List of Kubernetes template file paths to glob and parse.",
				ElementType: types.StringType,
			},
			function.MapParameter{
				Name:        "variables",
				Description: "Map of variables to use in the templates.",
				ElementType: types.StringType,
			},
		},
		Return: function.MapReturn{
			ElementType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"full_manifest": types.DynamicType,
					"source": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"file": types.StringType,
							"line": types.NumberType,
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
				},
			},
		},
		Summary: "Parse Kubernetes template files",
		Description: `Glob and split a list of Kubernetes template files. 
		Then if the variables map is set, parse the templates with the variables.
		Finally, return the parsed manifest as a map of objects with apiVersion, kind, metadata, manifest and source fields`,
	}
}

func (f *FunctionKubeParseTemplateFiles) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	// Retrieve the first argument as []string
	var templateFiles []string
	funcErr := req.Arguments.GetArgument(ctx, 0, &templateFiles)
	if funcErr != nil {
		resp.Error = function.NewArgumentFuncError(0, "Failed to parse 'template_files' as a list of strings: "+funcErr.Error())
		return
	}

	// Retrieve the second argument as map[string]any
	var values map[string]any
	funcErr = req.Arguments.GetArgument(ctx, 1, &values)
	if funcErr != nil {
		resp.Error = function.NewArgumentFuncError(1, "Failed to parse 'values' as a map: "+funcErr.Error())
		return
	}

	//call kube.ParseManifestTemplates
	parsedDocs, err := kube.ParseManifestTemplates(templateFiles, values)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "Failed to parse template files: "+funcErr.Error())
		return
	}

	// Convert the parsed documents to a map of objects

	objectAttr := map[string]attr.Type{
		"full_manifest": types.StringType,
		"source": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"file": types.StringType,
				"line": types.NumberType,
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

	parsedDocsMap := make(map[string]attr.Value)
	var diags diag.Diagnostics
	for _, doc := range parsedDocs {
		value := map[string]attr.Value{
			"api_version": types.StringValue(doc.APIVersion),
			"kind":        types.StringValue(doc.Kind),
		}
		value["source"], diags = basetypes.NewObjectValue(
			map[string]attr.Type{
				"file": types.StringType,
				"line": types.NumberType,
			},
			map[string]attr.Value{
				"file": types.StringValue(doc.Source.File),
				"line": types.NumberValue(big.NewFloat(float64(doc.Source.Line))),
			},
		)
		if diags.HasError() {
			txtError := "TODO"
			resp.Error = function.NewArgumentFuncError(0, "Failed to create source object: "+txtError)
		}
		value["metadata"], diags = basetypes.NewObjectValue(
			map[string]attr.Type{
				"name":      types.StringType,
				"namespace": types.StringType,
			},
			map[string]attr.Value{
				"name":      types.StringValue(doc.Metadata.Name),
				"namespace": types.StringValue(doc.Metadata.Namespace),
			},
		)
		if diags.HasError() {
			txtError := "TODO"
			resp.Error = function.NewArgumentFuncError(0, "Failed to create metadata object: "+txtError)
		}
		value["full_manifest"] = basetypes.NewDynamicNull() //TODO

		object, diags := basetypes.NewObjectValue(objectAttr, value)
		if diags.HasError() {
			txtError := "TODO"
			resp.Error = function.NewArgumentFuncError(0, "Failed to create parsed document object: "+txtError)
		}
		parsedDocsMap[doc.Metadata.Name] = object
	}
	result, diags := basetypes.NewMapValue(
		types.ObjectType{
			AttrTypes: objectAttr,
		},
		parsedDocsMap,
	)
	if diags.HasError() {
		txtError := "TODO"
		resp.Error = function.NewArgumentFuncError(0, "Failed to create result map: "+txtError)
		return
	}
	resp.Result.Set(ctx, result)
}
