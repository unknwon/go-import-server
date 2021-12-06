package main

import (
	"embed"
)

//go:embed templates/*.tmpl
var templates embed.FS
