package main

import (
	"fmt"
	"html/template"
	"os"
	"strings"

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

func Render(folder string, data []*Result) error {
	indexTmpl := template.Must(template.New("index").Parse(index))
	projectTmpl := template.Must(template.New("project").Parse(project))
	for _, res := range data {
		if len(res.Results) == 0 {
			continue
		}
		res.Filename = fmt.Sprintf("%s.html", strings.ToLower(res.Name))
		f, err := os.Create(fmt.Sprintf("%s/%s", folder, res.Filename))
		if err != nil {
			return err
		}
		defer f.Close()

		err = projectTmpl.Execute(f, res)
		if err != nil {
			return err
		}
	}
	indexPath := fmt.Sprintf("%s/index.html", folder)
	f, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return indexTmpl.Execute(f, data)
}

type ProjectResult struct {
	Name    string
	URL     string
	Results chan *checks.Result
}

type IndexData struct {
	Name     string
	Filename string
	Results  int
}

func GoRender(folder string, projectResults <-chan *ProjectResult) {
	indexTmpl := template.Must(template.New("index").Parse(`<html><head><title>GitScan Report</title></head><body><h1>GitScan Report</h1><ul>
{{range .}}{{if .Results}}<li><a href="{{.Filename}}">{{.Name}}</a> ({{.Results}} results)</li>{{end}}{{end}}
</ul></body></html>`))
	projectTmpl := template.Must(template.New("project").Parse(project))
	go func(folder string, projectResults <-chan *ProjectResult) {
		var data []*IndexData
		for projRes := range projectResults {
			project := &IndexData{
				Name:     projRes.Name,
				Filename: fmt.Sprintf("%s.html", strings.ToLower(projRes.Name)),
			}
			data = append(data, project)
			results := Result{
				Name:     project.Name,
				Filename: project.Filename,
			}
			for res := range projRes.Results {
				results.Results = append(results.Results, res)
				project.Results++
				f, err := os.Create(fmt.Sprintf("%s/%s", folder, project.Filename))
				if err != nil {
					continue
				}

				err = projectTmpl.Execute(f, results)
				f.Close()
				if err != nil {
					continue
				}

				indexPath := fmt.Sprintf("%s/index.html", folder)
				f, err = os.Create(indexPath)
				if err != nil {
					continue
				}
				indexTmpl.Execute(f, data)
				f.Close()
			}
		}
	}(folder, projectResults)
}
