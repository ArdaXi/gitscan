package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ardaxi/gitscan/database"
	"github.com/ardaxi/gitscan/providers"
	"github.com/gorilla/mux"
)

var db *database.DB
var baseURL string

func serve(mydb *database.DB, mybaseurl string) {
	db = mydb
	baseURL = mybaseurl
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/projects/index", indexHandler).Methods("GET").Queries("token", "{token}")
	r.HandleFunc("/projects/{id:[0-9]+}.html", projectHandler).Methods("GET").Queries("token", "{token}")
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8000", r))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html><head><title>GitScan</title></head><body>
	<form action="/projects/index" method="get"><label for="token">Token:</token>
	<input type="text" id="token" name="token"><button type="submit">Login</button>
	</form></body></html>`))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "<html><head><title>GitScan report</title></head><body>Logging in...<br>")
	vars := mux.Vars(r)
	token := vars["token"]
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: token,
		URL:   baseURL,
	})
	if err != nil {
		log.Printf("Error occurred logging into Gitlab: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to login to GitLab"))
		return
	}

	fmt.Fprintf(w, "Logged in as %s<br><br>", provider.Username())
	flush(w)

	projects := provider.ListAllProjects()
	for project := range projects {
		count, _ := db.ResultCount(project.ID())
		log.Printf("%s: %d", project.Name(), count)
		if count == 0 {
			continue
		}

		fmt.Fprintf(w, `<a href="/projects/%d.html?token=%s">%s</a> (%d results)<br>`, project.ID(), token, project.Name(), count)
		flush(w)
	}

	fmt.Fprintln(w, "</body></html>")
}

func projectHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "<html><head><title>GitScan report</title></head><body>Logging in...<br>")
	vars := mux.Vars(r)
	token := vars["token"]
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: token,
		URL:   baseURL,
	})

	if err != nil {
		log.Printf("Error occurred logging into Gitlab: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to login to GitLab"))
		return
	}

	fmt.Fprintf(w, "Logged in as %s<br><br>", provider.Username())
	flush(w)

	id, _ := strconv.Atoi(vars["id"])
	project, err := provider.GetProject(id)
	if err != nil {
		log.Printf("Error occurred getting project (%s): %s", vars["id"], err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to get project."))
		return
	}

	results, err := db.GetResults(project.ID())
	if err != nil {
		log.Printf("Error occurred getting results for project (%s): %s", vars["id"], err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to get results."))
		return
	}

	fmt.Fprintf(w, "<h1>%s - GitScan Report</h1><ul>", project.Name())
	for result := range results {
		fmt.Fprintf(w, `<li><a href="%s/blob/%s/%s">%s</a><br>%s<br>%s</li>`,
			project.URL(),
			result.Commit,
			result.Path,
			result.Path,
			result.Caption,
			result.Description,
		)
		flush(w)
	}

	fmt.Fprintln(w, "</ul></body></html>")
}

func flush(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
