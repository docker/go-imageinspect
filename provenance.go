package imageinspect

type Provenance struct { // TODO: this is only a stub, to be refactored later
	BuildSource     string            `json:",omitempty"`
	BuildDefinition string            `json:",omitempty"`
	BuildParameters map[string]string `json:",omitempty"`
	Materials       []Material
}

type Material struct {
	Type  string `json:",omitempty"`
	Ref   string `json:",omitempty"`
	Alias string `json:",omitempty"`
	Pin   string `json:",omitempty"`
}
