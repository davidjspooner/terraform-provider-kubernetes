// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/pmodel"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FileManifests{}

func NewFileManifests() datasource.DataSource {
	r := &FileManifests{
		prometheusTypeNameSuffix: "_manifest_files",
	}
	return r
}

// FileManifests defines the resource implementation.
type FileManifests struct {
	provider                 *KubernetesProvider
	prometheusTypeNameSuffix string
}

// FileManifestsModel describes the resource data model.
type FileManifestsModel struct {
	FileNames pmodel.Files `tfsdk:"file_data"`
	Documents types.Map    `tfsdk:"documents"`
}

func (r *FileManifests) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.prometheusTypeNameSuffix
}

func (r *FileManifests) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read yaml from a list of files and return all the inner documents",

		Attributes: map[string]schema.Attribute{
			"file_data": pmodel.FileListSchema(true),
			"documents": pmodel.ManifestMapSchema(),
		},
	}
}

func (r *FileManifests) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	r.provider, ok = req.ProviderData.(*KubernetesProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubernetesProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
}

func (r *FileManifests) readDocumentsFromReader(_ context.Context, reader io.Reader, _ *FileManifestsModel) []string {
	scanner := bufio.NewScanner(reader)
	var document bytes.Buffer
	var documents []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if document.Len() >= 0 {
				documents = append(documents, document.String())
			}
			document.Reset()
			continue
		}
		document.WriteString(line)
		document.WriteString("\n")
	}
	if document.Len() >= 0 {
		documents = append(documents, document.String())
	}
	return documents
}

func (r *FileManifests) expandTemplate(templateString string, values map[string]string) (string, error) {
	t := template.New("file")
	t, err := t.Parse(templateString)
	if err != nil {
		return "", err
	}
	var expanded bytes.Buffer
	err = t.Execute(&expanded, values)
	if err != nil {
		return "", err
	}
	return expanded.String(), nil
}

func (r *FileManifests) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileManifestsModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	sm := pmodel.StringMap{}
	sm.AddFileModel(&config.FileNames)

	var values map[string]string
	var diags diag.Diagnostics
	diags = config.FileNames.Values.ElementsAs(ctx, &values, false)
	resp.Diagnostics.Append(diags...)

	allDocuments := make(map[string]attr.Value)

	if !resp.Diagnostics.HasError() {
		sm.ForEach(ctx, func(filename string, content string) error {
			var err error
			if len(values) > 0 {
				content, err = r.expandTemplate(content, values)
				if err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Failed to expand file %s", filename),
						err.Error(),
					)
					return err
				}
			}

			sr := strings.NewReader(content)
			documents := r.readDocumentsFromReader(ctx, sr, &config)
			for _, document := range documents {

				manifest, err := pmodel.ReadManifest(document)
				if err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Failed to parse yaml from file %s", filename),
						err.Error(),
					)
					continue
				}
				key := manifest.Key()
				allDocuments[key], diags = types.ObjectValueFrom(ctx,pmodel.ManifestType, manifest)
			}
			return nil
		})
	}

	if resp.Diagnostics.HasError() {

	}

	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
