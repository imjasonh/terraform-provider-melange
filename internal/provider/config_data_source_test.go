// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.melange_config.minimal", "package.name", "minimal"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "package.version", "0.0.1"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "package.description", "a very basic melange example"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "environment.contents.repositories.#", "1"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "environment.contents.repositories.0", "https://dl-cdn.alpinelinux.org/alpine/edge/main"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "environment.contents.packages.#", "2"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "environment.contents.packages.0", "alpine-baselayout-data"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "environment.contents.packages.1", "busybox"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "pipeline.#", "1"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "pipeline.0.runs", "echo \"hello\"\n"),
			),
		}},
	})
}
