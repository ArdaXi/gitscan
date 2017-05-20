package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ardaxi/go-gitlab"
)

var token = flag.String("token", "", "GitLab API token")
var baseurl = flag.String("url", "https://gitlab.com/api/v3/", "GitLab base URL")
var signaturePath = flag.String("signatures", "signatures.json", "Path to signatures file")

func main() {
	flag.Parse()

	log.Printf("Parsing signatures from %v", *signaturePath)
	signatures, err := ParseSignatures(*signaturePath)
	handleError(err, "parse signatures")

	_ = signatures

	git := gitlab.NewClient(nil, *token)
	err = git.SetBaseURL(*baseurl)
	handleError(err, "set base URL")

	log.Printf("Logging in to GitLab at %v", *baseurl)
	user, _, err := git.Users.CurrentUser()
	handleError(err, "get current user")

	log.Printf("Logged in as %s", user.Username)

	var projects []*gitlab.Project

	if user.IsAdmin {
		log.Println("User is admin, checking all projects")
		projects, _, err = git.Projects.ListAllProjects(nil)
		if err != nil {
			log.Printf("Failed to list all projects: %s", err)
			log.Println("Falling back to checking user projects")
			projects, _, err = git.Projects.ListProjects(nil)
		}
	} else {
		log.Println("User is not admin, checking user projects")
		projects, _, err = git.Projects.ListProjects(nil)
	}
	handleError(err, "get projects")

	var allResults []*Result

	for _, project := range projects {
		log.Printf("Scanning project %v", project.Name)
		tree, _, err := git.Repositories.ListTree(project.ID, &gitlab.ListTreeOptions{Recursive: gitlab.Bool(true)})
		if err != nil {
			log.Printf("Couldn't retrieve tree for %s: %s", project.Name, err)
			continue
		}

		projectResult := &Result{
			Name: project.Name,
			URL:  project.WebURL,
		}

		for _, v := range tree {
			if v.Type != "tree" {
				count, results := CheckPath(signatures, v.Path)
				projectResult.Count += count
				projectResult.CheckResults = append(projectResult.CheckResults, results...)
			}
		}

		allResults = append(allResults, projectResult)
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
