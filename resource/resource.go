package resource

import (
	"embed"
)

//go:embed templates
var Fs embed.FS

const TemplateFeaturesPath = "templates/features"
