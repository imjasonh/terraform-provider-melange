# TODO: `terraform-provider-melange`

### Installing

```
terraform {
  required_providers {
    melange = { source  = "imjasonh/melange" }
  }
}
```

Then `terraform init -upgrade`.

(This is not yet published; it will be published under `chainguard-dev` when it's ready)

### Build a single package

```
provider "melange" {
    archs              = ["x86_64", "aarch64"]
    extra_repositories = ["https://packages.wolfi.dev/os"]
    extra_keyring      = ["https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"]
}

data "melange_config" "config" {
    config_contents = file("package.yaml")
}

resource "melange_build" "package" {
    config          = data.melange_config.config.config
    config_contents = data.melange_config.config.config_contents
}
```

After applying this config, `packages/$ARCH/package-0.0.1-rX.apk` will be built, and `packages/$ARCH/APKINDEX.tar.gz` will be updated.

(This passes locally but currently fails in CI...)

### Build a graph of inter-dependent Melange configs

```
data "melange_graph" "graph" {
    files = fileset(path.module, "*.yaml")
}

resource "melange_build" "foo" {
    for_each = data.melange_graph.graph.configs
    depends_on = data.melange_graph.graph.deps[each.key]

    config          = each.key
    config_contents = data.melange_graph.graph[each.key].config_contents
}
```

This will crawl a collection of Melange config files, construct a graph of the order they should be built to ensure dependencies are met, and build them in that order with Terraform's configured concurrency.

(This is not yet implemented)

### Build a package locally, then build it into an image using `apko_build`

```
data "melange_config" "config" {
    config_contents = file("package.yaml")
}

resource "melange_build" "package" {
    config          = data.melange_config.config.config
    config_contents = data.melange_config.config.config_contents
}

provider "apko" {
    archs              = ["x86_64", "aarch64"]
    extra_repositories = ["https://packages.wolfi.dev/os", "./packages"]
    extra_keyring      = ["https://packages.wolfi.dev/os/wolfi-signing.rsa.pub", "./local-signing.rsa.pub"]
}

data "apko_config" "config" {
    config_contents = file("image.yaml")
    extra_packages  = [data.melange_config.config.config.package.name]
}

resource "apko_build" "image" {
    config = data.apko_config.config.config
}
```

(This is not yet tested)
