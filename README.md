# TODO

```
provider "melange" {
    archs              = ["x86_64", "aarch64"]
    extra_repositories = ["https://packages.wolfi.dev/os"]
    extra_keyring      = ["https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"]
}

data "melange_graph" "graph" {
    files = fileset(path.module, "*.yaml")
}

resource "melange_build" "foo" {
    for_each = data.melange_graph.graph.configs
    depends_on = data.melange_graph.graph.deps[each.key]

    config = each.key
}
```
