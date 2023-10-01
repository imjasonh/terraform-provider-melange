package provider

import apko_types "chainguard.dev/apko/pkg/build/types"

type Package struct {
	// The name of the package
	Name string `yaml:"name"`
	// The version of the package
	Version string `yaml:"version"`
	// The monotone increasing epoch of the package
	Epoch uint32 `yaml:"epoch"`
}

// The root melange configuration
type Configuration struct {
	// Package metadata
	Package Package `yaml:"package"`
	// The specification for the packages build environment
	Environment apko_types.ImageConfiguration
}
