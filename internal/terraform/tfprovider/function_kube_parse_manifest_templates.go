package tfprovider

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/generic/kube"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var manifestMapElementAttrType = map[string]attr.Type{
	"text": types.StringType,
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

// FunctionKubeParseTemplateFiles implements the terraform function.Function interface.
type FunctionKubeParseTemplateFiles struct {
}

// NewFunctionKubeParseTemplateFiles creates a new instance of FunctionKubeParseTemplateFiles.
func NewFunctionKubeParseTemplateFiles() function.Function {
	return &FunctionKubeParseTemplateFiles{}
}

// Metadata provides metadata for the function.
func (f *FunctionKubeParseTemplateFiles) Metadata(ctx context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "kube_manifest_files"
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
		Return: function.DynamicReturn{
			// ElementType: types.ObjectType{
			// 	AttrTypes: map[string]attr.Type{
			// 		"full_manifest": types.DynamicType,
			// 		"source": types.ObjectType{
			// 			AttrTypes: map[string]attr.Type{
			// 				"file": types.StringType,
			// 				"line": types.NumberType,
			// 			},
			// 		},
			// 		"api_version": types.StringType,
			// 		"kind":        types.StringType,
			// 		"metadata": types.ObjectType{
			// 			AttrTypes: map[string]attr.Type{
			// 				"name":      types.StringType,
			// 				"namespace": types.StringType,
			// 			},
			// 		},
			// 	},
			// },
		},
		Summary: "Parse Kubernetes template files",
		Description: `Glob and split a list of Kubernetes template files. 
		Then if the variables map is set, parse the templates with the variables.
		Finally, return the parsed manifest as a map of objects with apiVersion, kind, metadata, manifest and source fields`,
	}
}

func SplitAndExpandTemplateFiles(filenames []string, variables map[string]any) (types.Map, diag.Diagnostics) {
	//call kube.ParseManifestTemplates

	var result types.Map
	var diags diag.Diagnostics

	parsedDocs, err := kube.SplitAndExpandTemplates(filenames, variables)
	if err != nil {
		diags.AddError("Error reading files", err.Error())
		return result, diags
	}
	if len(parsedDocs) == 0 {
		details := fmt.Sprintf("No documents found matching %s", strings.Join(filenames, ", "))
		diags.AddError("No documents found", details)
		return result, diags
	}

	// Convert the parsed documents to a map of objects

	parsedDocsMap := make(map[string]attr.Value)
	for _, doc := range parsedDocs {
		value := map[string]attr.Value{
			"api_version":   types.StringValue(doc.APIVersion),
			"kind":          types.StringValue(doc.Kind),
			"manifest_text": types.StringValue(doc.Manifest),
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
			return result, diags
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
			return result, diags
		}

		object, diags := basetypes.NewObjectValue(manifestMapElementAttrType, value)
		if diags.HasError() {
			return result, diags
		}
		key := fmt.Sprintf("%s:%s:%s", doc.Kind, doc.Metadata.Namespace, doc.Metadata.Name)
		_, exists := parsedDocsMap[key]
		if exists {
			diags.AddError("Duplicate manifest", fmt.Sprintf("Duplicate manifest found for kind=%q namespace=%q name=%q at %s [%d]", doc.Kind, doc.Metadata.Namespace, doc.Metadata.Name, doc.Source.File, doc.Source.Line))
			// Add the existing object to the diagnostics
			return result, diags
		}
		parsedDocsMap[key] = object
	}
	result, diags = basetypes.NewMapValue(
		types.ObjectType{
			AttrTypes: manifestMapElementAttrType,
		},
		parsedDocsMap,
	)
	return result, nil
}

func (f *FunctionKubeParseTemplateFiles) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	// Retrieve the first argument as []string
	var filenames []string
	funcErr := req.Arguments.GetArgument(ctx, 0, &filenames)
	if funcErr != nil {
		resp.Error = function.NewArgumentFuncError(0, "Failed to parse 'template_files' as a list of strings: "+funcErr.Error())
		return
	}

	// Retrieve the second argument as map[string]any
	var variables map[string]any
	funcErr = req.Arguments.GetArgument(ctx, 1, &variables)
	if funcErr != nil {
		resp.Error = function.NewArgumentFuncError(1, "Failed to parse 'values' as a map: "+funcErr.Error())
		return
	}

	result, diags := SplitAndExpandTemplateFiles(filenames, variables)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
	}
	resp.Result.Set(ctx, result)
}
