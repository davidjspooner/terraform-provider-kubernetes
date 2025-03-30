// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"text/template"

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
	r := &FileManifests{}
	return r
}

// FileManifests defines the resource implementation.
type FileManifests struct {
	provider *KubernetesProvider
}

// FileManifestsModel describes the resource data model.
type FileManifestsModel struct {
	Filenames types.List `tfsdk:"filenames"`
	Values    types.Map  `tfsdk:"values"`
	Documents types.Set  `tfsdk:"documents"`
}

func (r *FileManifests) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest_files"
}

func (r *FileManifests) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Read a list of yaml and return all the inner documents",

		Attributes: map[string]schema.Attribute{
			"filenames": schema.ListAttribute{
				MarkdownDescription: "Filenames to read",
				ElementType:         types.StringType,
				Required:            true,
			},
			"values": schema.MapAttribute{
				MarkdownDescription: "If defined, treat files as goloang templates and render them with these values",
				ElementType:         types.StringType,
			},
			"documents": schema.SetAttribute{
				MarkdownDescription: "Set of documents",
				ElementType:         types.StringType,
				Computed:            true,
			},
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

func (r *FileManifests) readDocumentsFromReader(_ context.Context, reader io.Reader, _ *FileManifestsModel) ([]string, error) {
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
	return documents, nil
}

func (r *FileManifests) readDocumentsFromFile(ctx context.Context, filename string, config *FileManifestsModel) ([]string, error) {
	// Read the file
	values := config.Values.Elements()
	if len(values) == 0 {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}

		defer f.Close()
		return r.readDocumentsFromReader(ctx, f, config)
	}
	// Render the file as a golang template
	t := template.New("file")
	t, err := t.ParseFiles(filename)
	if err != nil {
		return nil, err
	}
	expanded := bytes.Buffer{}
	err = t.ExecuteTemplate(&expanded, filename, values)
	if err != nil {
		return nil, err
	}
	return r.readDocumentsFromReader(ctx, &expanded, config)
}

func (r *FileManifests) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileManifestsModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	filenames := config.Filenames.Elements()
	allDocuments := make([]attr.Value, 0, len(filenames))

	for n := range filenames {
		filename := filenames[n].String()
		stats, err := os.Stat(filename)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to stat file %s", filename),
				err.Error(),
			)
			continue
		}
		if stats.IsDir() {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to read file %s", filename),
				"Is a directory",
			)
		} else {
			documents, err := r.readDocumentsFromFile(ctx, filename, &config)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Failed to read file %s", filename),
					err.Error(),
				)
				continue
			}
			for _, document := range documents {
				allDocuments = append(allDocuments, types.StringValue(document))
			}
		}
	}

	var diags diag.Diagnostics
	config.Documents, diags = types.SetValue(types.StringType, allDocuments)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
