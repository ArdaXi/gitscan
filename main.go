package main

import (
	"flag"
	"log"

	"github.com/ardaxi/gitscan/checks"
	"github.com/ardaxi/gitscan/database"
	"github.com/ardaxi/gitscan/providers"
)

var token = flag.String("token", "", "GitLab API token")
var baseurl = flag.String("url", "https://gitlab.com/api/v3/", "GitLab base URL")
var signaturePath = flag.String("signatures", "signatures.json", "Path to signatures file")
var limit = flag.Int("limit", -1, "Amount of repositories to scan")
var dsn = flag.String("dsn", "dname=scan", "PostgreSQL DSN")
var server = flag.Bool("server", false, "Run GitScan server")

func main() {
	flag.Parse()

	log.Println("Connecting to database")
	db, err := database.Connect(*dsn)
	handleError(err, "connect to database")

	_ = db

	if *server {
		serve(db, *baseurl)
		return
	}

	log.Printf("Logging into Gitlab at %s", *baseurl)
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: *token,
		URL:   *baseurl,
	})
	handleError(err, "start provider")

	log.Printf("Parsing signatures from %v", *signaturePath)
	err = checks.ParseSignatures(*signaturePath)
	handleError(err, "parse signatures")

	projects := provider.ListAllProjects()
	for project := range projects {
		log.Printf("Looking up project %v in DB", project.Name())
		commitID, err := project.LastCommit()
		if err != nil {
			log.Printf("Could not find last commit for project %s", project.Name())
			continue
		}

		lastScanned, err := db.GetLastScanned(project.ID())
		if err != nil {
			log.Printf("Couldn't find or create %s in DB: %s", project.Name(), err)
		}

		if lastScanned == commitID {
			log.Printf("Already scanned commit %s of project %s", commitID, project.Name())
			continue
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

		resultCh := make(chan *database.Result)

		go func(resultCh <-chan *database.Result) {
			for result := range resultCh {
				db.AddResult(result)
			}
		}(resultCh)

		for _, f := range files {
			for _, check := range checks.Checks {
				results := check(f)
				for _, result := range results {
					resultCh <- &database.Result{
						Project:     project.ID(),
						Commit:      commitID,
						Path:        result.File.Path(),
						Caption:     result.Caption,
						Description: result.Description,
					}
				}
			}
		}

		close(resultCh)

		err = db.SetLastScanned(project.ID(), commitID)
		if err != nil {
			log.Printf("Failed to set last scanned commit ID: %s", err)
		}

		if *limit == 0 {
			break
		}
	}
}

func handleError(err error, what string) {
	if err != nil {
		log.Fatalf("Failed to %s: %s", what, err)
	}
}
