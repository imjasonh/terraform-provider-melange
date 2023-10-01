// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"testing"

	apko_types "chainguard.dev/apko/pkg/build/types"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.alpinelinux.org/alpine/go/repository"
)

func TestMain(m *testing.M) {
	// Delete any existing packages before running tests, to ensure
	// the APKINDEX is correct and builds actually happen.
	if err := os.RemoveAll("packages"); err != nil {
		log.Fatalf("failed to remove packages: %v", err)
	}
	os.Exit(m.Run())
}

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"melange": providerserver.NewProtocol6WithError(&Provider{
		archs: []string{runtime.GOARCH},
	}),
}

var arch = apko_types.ParseArchitecture(runtime.GOARCH).ToAPK()

func TestAccBuildResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{{
			Config: `
data "melange_config" "minimal" {
	config_contents = file("${path.module}/testdata/minimal.yaml")
}

resource "melange_build" "build" {
	config          = data.melange_config.minimal.config
	config_contents = data.melange_config.minimal.config_contents
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("melange_build.build", "config.package.name", "minimal"),
				//resource.TestCheckResourceAttr("melange_build.build", "id", "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"),
			),
		}},
	})

	// Check the apk.
	fn := fmt.Sprintf("packages/%s/minimal-0.0.1-r3.apk", arch)
	f, err := os.Open(fn)
	if err != nil {
		t.Fatalf("failed to open apk: %v", err)
	}
	defer f.Close()
	pkg, err := repository.ParsePackage(f)
	if err != nil {
		t.Fatalf("failed to parse apk: %v", err)
	}
	if pkg.Name != "minimal" {
		t.Errorf("unexpected package name: %v", pkg.Name)
	}
	if pkg.Version != "0.0.1-r3" {
		t.Errorf("unexpected package version: %v", pkg.Version)
	}

	// Check the index.
	fn = fmt.Sprintf("packages/%s/APKINDEX.tar.gz", arch)
	f, err = os.Open(fn)
	if err != nil {
		t.Fatalf("failed to open index: %v", err)
	}
	defer f.Close()
	idx, err := repository.IndexFromArchive(f)
	if err != nil {
		t.Fatalf("failed to parse index: %v", err)
	}
	if len(idx.Packages) != 1 {
		t.Errorf("unexpected number of packages in index: %v", len(idx.Packages))
	}
	if string(idx.Packages[0].Checksum) != string(pkg.Checksum) {
		t.Errorf("checksum mismatch: %v != %v", idx.Packages[0].Checksum, pkg.Checksum)
	}
	// TODO(jason): Check that index is signed with the key.
}
