package resume

// Metadata is generated from mygo.json, the application identity source.
type Metadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Resources contains immutable resume definitions, independent of disk paths.
type Resources struct {
	TemplateHTML string
	DocumentMD   string
	Schema       []FieldMeta
	Metadata     Metadata
}

// StoragePaths separates writable user data from bundled resources.
type StoragePaths struct {
	ContentFile string
	OutputDir   string
}
