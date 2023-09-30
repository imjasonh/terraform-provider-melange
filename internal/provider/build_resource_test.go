// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBuildResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"melange": providerserver.NewProtocol6WithError(&Provider{
				archs: []string{runtime.GOARCH},
			}),
		},
		Steps: []resource.TestStep{{
			// based on https://github.com/chainguard-dev/melange/blob/main/examples/minimal.yaml
			Config: `
data "melange_config" "minimal" {
	config_contents = <<EOF
package:
  name: minimal
  version: 0.0.1
  description: a very basic melange example
environment:
  contents:
    repositories:
      - https://dl-cdn.alpinelinux.org/alpine/edge/main
    packages:
      - alpine-baselayout-data
      - busybox
pipeline:
  - runs: echo "hello"
EOF
}

resource "melange_build" "build" {
	config = data.melange_config.minimal.config
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("melange_build.build", "config.package.name", "minimal"),
				resource.TestCheckResourceAttr("melange_build.build", "id", "minimal-0.0.1-r0"),
			),
		}},
	})
}
