package main

import "embed"

//go:embed index.html app.js styles.css
var embeddedFiles embed.FS

var (
	indexHTML  = mustReadAsset("index.html")
	appJS     = mustReadAsset("app.js")
	stylesCSS = mustReadAsset("styles.css")
)

func mustReadAsset(path string) []byte {
	data, err := embeddedFiles.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
