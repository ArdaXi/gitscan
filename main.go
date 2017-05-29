package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ardaxi/gitscan/checks"
	"github.com/ardaxi/gitscan/database"
	"github.com/ardaxi/gitscan/providers"
)

var token = flag.String("token", "", "GitLab API token")
var baseurl = flag.String("url", "https://gitlab.com/api/v3/", "GitLab base URL")
var signaturePath = flag.String("signatures", "signatures.json", "Path to signatures file")
var limit = flag.Int("limit", -1, "Amount of repositories to scan")
var dsn = flag.String("dsn", "dname=scan", "PostgreSQL DSN")

func main() {
	flag.Parse()

	log.Println("Connecting to database")
	db, err := database.Connect(*dsn)
	handleError(err, "connect to database")

	log.Printf("Parsing signatures from %v", *signaturePath)
	err = checks.ParseSignatures(*signaturePath)
	handleError(err, "parse signatures")

	log.Printf("Logging into Gitlab at %s", *baseurl)
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: *token,
		URL:   *baseurl,
	})
	handleError(err, "start provider")

	var allResults []*Result

	folder := fmt.Sprintf("result-%s", time.Now().Format("20060102-1504"))
	_ = os.Mkdir(folder, os.ModePerm)
	indexPath := fmt.Sprintf("%s/index.html", folder)

	wd, err := os.Getwd()
	handleError(err, "get current directory")
	log.Printf("Rendering results to %s", filepath.Join(wd, indexPath))

	projects := provider.ListAllProjects()
	for project := range projects {
		log.Printf("Looking up project %v in DB", project.Name())
		commitID := project.LastCommit()
		lastScanned, err := db.GetLastScanned(project.ID())
		if err != nil {
			log.Printf("Couldn't find or create %s in DB.", project.Name())
		}
		log.Printf("Scanning project %v", project.Name())
		files, err := project.Files()
		if err != nil {
			log.Printf("Couldn't retrieve tree for %s: %s", project.Name(), err)
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
			for _, check := range checks.Checks {
				results := check(f)
				projectResult.Results = append(projectResult.Results, results...)
			}
		}

		allResults = append(allResults, projectResult)

		Render(folder, allResults)

		if *limit == 0 {
			break
		}
	}

	err = Render(folder, allResults)
	handleError(err, "render results")

	log.Printf("Rendered results to %s", filepath.Join(wd, indexPath))
}

func handleError(err error, what string) {
	if err != nil {
		log.Fatalf("Failed to %s: %s", what, err)
	}
}
