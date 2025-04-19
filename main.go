// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/davidjspooner/terraform-provider-kubernetes/internal/tfprovider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run "go generate" to format example terraform files and generate the docs for the registry/website

// If you do not have terraform installed, you can remove the formatting command, but its suggested to
// ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name kubernetes

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	Version string = "dev"
	BuiltAt string = "unknown"
	Commit  string = "unknown"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool
	var doVersion bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.BoolVar(&doVersion, "version", false, "print the version and exit")
	flag.Parse()

	if doVersion {
		fmt.Printf("Version : %s\nBuilt at: %s\nCommit  : %s\n", Version, BuiltAt, Commit)
		os.Exit(0)
	}

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		// Also update the tfplugindocs generate command to either remove the
		// -provider-name flag or set its value to the updated provider name.
		Address: "dstower.home.dolbyn.com/davidjspooner/kubernetes",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), tfprovider.NewProvider(Version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
