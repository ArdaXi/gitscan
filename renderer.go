package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/ardaxi/gitscan/checks"
)

type Result struct {
	Filename string
	Name     string
	URL      string
	Results  []*checks.Result
}

const index = `<html><head><title>GitScan Report</title></head><body><h1>GitScan Report</h1><ul>
{{range .}}{{if len .Results}}<li><a href="{{.Filename}}">{{.Name}}</a> ({{len .Results}} results)</li>{{end}}{{end}}
</ul></body></html>`

const project = `<html><head><title>{{.Name}} - GitScan Report</title></head><body><h1>{{.Name}} - GitScan Report</h1><ul>
{{range .Results}}<li><a href="{{$.URL}}/blob/master/{{.File.Path}}">{{.File.Path}}</a><br />{{.Caption}}{{with .Description}}<br />{{.}}{{end}}</li>{{end}}
</ul></body></html>`

func Render(data []*Result) (string, error) {
	folder := fmt.Sprintf("result-%s", time.Now().Format("20060102-1504"))
	_ = os.Mkdir(folder, os.ModePerm)
	indexTmpl := template.Must(template.New("index").Parse(index))
	projectTmpl := template.Must(template.New("project").Parse(project))
	for _, res := range data {
		if len(res.Results) == 0 {
			continue
		}
		res.Filename = fmt.Sprintf("%s.html", strings.ToLower(res.Name))
		f, err := os.Create(fmt.Sprintf("%s/%s", folder, res.Filename))
		if err != nil {
			return "", err
		}
		defer f.Close()

		err = projectTmpl.Execute(f, res)
		if err != nil {
			return "", err
		}
	}
	indexPath := fmt.Sprintf("%s/index.html", folder)
	f, err := os.Create(indexPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return indexPath, indexTmpl.Execute(f, data)
}
