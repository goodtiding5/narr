package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
	htemplate "html/template"
)

var code_template = `// autogenerated. do not edit!

package server

var assets_bundle = map[string]asset{
	{{- range .}}
	"{{.Name}}": {etag: "{{.Etag}}", body: "{{.Body}}"},
	{{- end }}
}

func init() {
	assets = assets_bundle
}
`

type asset struct {
	Name, Etag, Body string
}

func shasum(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func encode(b []byte) string {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(b)
	zw.Close()
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func main() {
	assets := make([]asset, 0)
	filepatterns := []string{
		"assets/graphicarts/*.svg",
		"assets/graphicarts/*.png",
		"assets/javascripts/*.js",
		"assets/stylesheets/*.css",
		"assets/stylesheets/*.map",
	}
	fmt.Printf("%8s %8s %s\n", "original", "encoded", "filename")
	for _, pattern := range filepatterns {
		filenames, _ := filepath.Glob(pattern)
		for _, filename := range filenames {
			content, _ := ioutil.ReadFile(filename)
			assets = append(assets, asset{
				Name: strings.TrimPrefix(filename, "assets/"),
				Etag: shasum(content),
				Body: encode(content),
			})
			fmt.Printf(
				"%8d %8d %s\n",
				len(content),
				len(assets[len(assets)-1].Body),
				filename,
			)
		}
	}
	var indexbuf bytes.Buffer
	htemplate.Must(htemplate.New("index.html").Delims("{%", "%}").Funcs(htemplate.FuncMap{
		"inline": func(svg string) htemplate.HTML {
			content, _ := ioutil.ReadFile("assets/graphicarts/" + svg)
			return htemplate.HTML(content)
		},
	}).ParseFiles("assets/index.html")).Execute(&indexbuf, nil)
	indexcontent := indexbuf.Bytes()
	assets = append(assets, asset{
		Name: "index.html",
		Etag: shasum(indexcontent),
		Body: encode(indexcontent),
	})

	var buf bytes.Buffer
	template := template.Must(template.New("code").Parse(code_template))
	template.Execute(&buf, assets)
	ioutil.WriteFile("server/assets_bundle.go", buf.Bytes(), 0644)
}
