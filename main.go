package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ardaxi/gitscan/providers"
)

var token = flag.String("token", "", "GitLab API token")
var baseurl = flag.String("url", "https://gitlab.com/api/v3/", "GitLab base URL")
var signaturePath = flag.String("signatures", "signatures.json", "Path to signatures file")
var limit = flag.Int("limit", -1, "Amount of repositories to scan")

func main() {
	flag.Parse()

	log.Printf("Parsing signatures from %v", *signaturePath)
	signatures, err := ParseSignatures(*signaturePath)
	handleError(err, "parse signatures")

	_ = signatures

	log.Printf("Logging into Gitlab at %s", *baseurl)
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: *token,
		URL:   *baseurl,
	})
	handleError(err, "start provider")

	var allResults []*Result

	projects := provider.ListAllProjects()
	for project := range projects {
		log.Printf("Scanning project %v", project.Name())
		files, err := project.Files()
		if err != nil {
			log.Printf("Couldn't retrieve tree for %s: %s", project.Name, err)
			continue
		}

		if *limit != -1 {
			*limit--
		}

		projectResult := &Result{
			Name: project.Name(),
			URL:  project.URL(),
		}

		for _, f := range files {
			count, results := CheckPath(signatures, f.Path())
			projectResult.Count += count
			projectResult.CheckResults = append(projectResult.CheckResults, results...)
		}

		allResults = append(allResults, projectResult)

		if *limit == 0 {
			break
		}
	}

	indexPath, err := Render(allResults)
	handleError(err, "render results")

	wd, err := os.Getwd()
	handleError(err, "get current directory")
	log.Printf("Rendered results to %s", filepath.Join(wd, indexPath))
}

func handleError(err error, what string) {
	if err != nil {
		log.Fatalf("Failed to %s: %s", what, err)
	}
}
