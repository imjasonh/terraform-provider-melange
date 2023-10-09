// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConfigDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			// based on https://github.com/chainguard-dev/melange/blob/main/examples/minimal.yaml
			Config: `
data "melange_config" "minimal" {
	config_contents = <<EOF
package:
  name: minimal
  version: 0.0.1
  epoch: 3
  description: a very basic melange example
environment:
  contents:
    packages:
      - wolfi-baselayout
      - busybox
pipeline:
  - runs: echo "hello"
EOF
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.package.name", "minimal"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.package.version", "0.0.1"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.package.epoch", "3"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.environment.contents.repositories.#", "1"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.environment.contents.repositories.0", "https://packages.wolfi.dev/os"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.environment.contents.packages.#", "2"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.environment.contents.packages.0", "wolfi-baselayout"),
				resource.TestCheckResourceAttr("data.melange_config.minimal", "config.environment.contents.packages.1", "busybox"),
			),
		}},
	})
}
