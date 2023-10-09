// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGraphDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"melange": providerserver.NewProtocol6WithError(&Provider{
				archs:        []string{runtime.GOARCH},
				dir:          "./testdata",
				repositories: []string{"https://packages.wolfi.dev/os"},
				keyring:      []string{"https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"},
			}),
		},
		Steps: []resource.TestStep{{
			Config: `data "melange_graph" "graph" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.%", "5"), // there should be a dep entry for every package.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.a.#", "1"),
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.a.0", "b"), // a depends on b.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.b.#", "2"),
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.b.0", "c"),       // b depends on c.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.b.1", "d"),       // b depends on d.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.c.#", "0"),       // c depends on nothing.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.d.#", "0"),       // d depends on nothing.
				resource.TestCheckResourceAttr("data.melange_graph.graph", "deps.minimal.#", "0"), // `minimal` depends on nothing.
				func(s *terraform.State) error {
					t.Logf("State: %+v", s)
					return nil
				},
			),
		}},
	})
}
