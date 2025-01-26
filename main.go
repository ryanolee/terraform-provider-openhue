package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/ryanolee/terraform-provider-talk/internal/provider"
)

func main() {
	opts := providerserver.ServeOpts{
		// TODO: Update this string with the namespace of your provider
		Address: "registry.terraform.io/ryanolee/openhue",
	}

	err := providerserver.Serve(context.Background(), provider.New, opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
