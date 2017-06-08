package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaxi/gitscan/database"
	"github.com/ardaxi/gitscan/providers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var db *database.DB
var baseURL string
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
var homeTempl = template.Must(template.New("").Parse(homeHTML))

const pongWait = 60 * time.Second

func serve(mydb *database.DB, mybaseurl string) {
	db = mydb
	baseURL = mybaseurl
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/ws", wsHandler).Methods("GET").Queries("token", "{token}")
	r.HandleFunc("/projects/index", indexHandler).Methods("GET").Queries("token", "{token}")
	r.HandleFunc("/projects/{id:[0-9]+}.html", projectHandler).Methods("GET").Queries("token", "{token}")
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8000", r))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	var v = struct {
		Host string
	}{
		Host: r.Host,
	}
	if err := homeTempl.Execute(w, &v); err != nil {
		log.Printf("Failed to execute template: %s", err)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Couldn't start ws session: %s", err)
		return
	}

	vars := mux.Vars(r)
	token := vars["token"]

	go writer(ws, token)
	reader(ws)
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, token string) {
	provider, err := providers.Providers["gitlab"](&providers.Options{
		Token: token,
		URL:   baseURL,
	})
	if err != nil {
		log.Printf("Error occurred logging into Gitlab: %s", err)
		if err := ws.WriteMessage(websocket.TextMessage, status("Failed to login")); err != nil {
			log.Printf("Error occurred writing error: %s", err)
		}
		return
	}
	if err := ws.WriteMessage(websocket.TextMessage, status("Logged in as %s", provider.Username())); err != nil {
		log.Printf("Error occurred writing error: %s", err)
	}

	projects := provider.ListAllProjects()
	for project := range projects {
		count, _ := db.ResultCount(project.ID())
		if count == 0 {
			continue
		}

		res := result(`<a href="/projects/%d.html?token=%s">%s</a> (%d results)`, project.ID(), token, project.Name(), count)
		if err := ws.WriteMessage(websocket.TextMessage, res); err != nil {
			log.Printf("Error occurred writing result: %s", err)
		}
	}
}

func status(msg string, a ...interface{}) []byte {
	return []byte(fmt.Sprintf(`{"html":"`+msg+`"}`, a...))
}

func result(msg string, a ...interface{}) []byte {
	safeMsg, _ := json.Marshal(msg)
	return []byte(fmt.Sprintf(`{"html":`+string(safeMsg)+`,"result":true}`, a...))
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

type wsMessage struct {
	HTML   string `json:"html"`
	Result bool   `json:"result"`
}

const homeHTML = `<!DOCTYPE html><html lang="en"><head>
<title>GitScan Report</title></head><body>
<form id="login-form"><label for="token">Token:</label>
<input type="text" id="token" name="token"><button type="submit">Login</button>
</form><br><div id="status"><div>Not logged in</div><br></div><br><div id="results"></div>
<script type="text/javascript">
    var addResult = function(results, data) {
		var result = document.createElement('div')
		result.innerHTML = data.html
		results.appendChild(result)
	}
    var connect = function(token) {
		var status = document.getElementById('status')
		var results = document.getElementById('results')
		var conn = new WebSocket("ws://{{.Host}}/ws?token="+token)
		conn.onclose = function(e) {
			newStatus = document.createElement('div')
			newStatus.textContent = "Connection closed"
			status.appendChild(newStatus)
		}
		conn.onmessage = function(e) {
			var data = JSON.parse(e.data)
			if (data.result) {
				addResult(results, data)
				return
			}
			newStatus = document.createElement('div')
			newStatus.textContent = data.html
			status.appendChild(newStatus)
		}
	}
    window.onload=function() {
		document.getElementById("login-form").onsubmit=function() {
			var tokenField = document.getElementById("token")
			connect(tokenField.value)
			return false
		}
	}
</script>
</body></html>
`
