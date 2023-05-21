package main

import (
	"fmt"
	"log"
	"pkg"
	"os"
	"html/template"
	"strings"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"github.com/qor/render"
	"github.com/gorilla/websocket"
)

var renderer *render.Render;
var upgrader = websocket.Upgrader{}
var hub = &pkg.Hub{CommandChSize: 100}
var jobIdRegex = regexp.MustCompile(`jobs/(\d+)`)

func defaultCtx() map[string]interface{} {
	ctx := make(map[string]interface{})
	if pkg.Env == "production" {
		snippet := `
		<!-- Google tag (gtag.js) -->
		<script async src="https://www.googletagmanager.com/gtag/js?id=ID"></script>
		<script>
			window.dataLayer = window.dataLayer || [];
			function gtag(){dataLayer.push(arguments);}
			gtag('js', new Date());

			gtag('config', 'ID');
		</script>
`
		snippet = strings.ReplaceAll(snippet, "ID", os.Getenv("GOOGLE_ANALYTICS_ID"))

		if pkg.Debug {
			fmt.Printf("snippet:\n %s\n", snippet)
		}
		
		ctx["googleAnalytics"] = template.HTML(snippet)
	}
	return ctx
}

func writeError(w http.ResponseWriter, msg string, errorNum int) {
	http.Error(w, msg, errorNum)
}

func writeInteralServerError(w http.ResponseWriter, r *http.Request, msg string) {
	fmt.Printf("Internal Server Error at %s: %s\n", r.URL.Path, msg)
	writeError(w, msg, http.StatusInternalServerError)
}

func writeNotFound(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Not Found at %s\n", r.URL.Path)
	writeError(w, "Not found", http.StatusNotFound)
}

func root(w http.ResponseWriter, r *http.Request) {
	ctx := defaultCtx()
	renderer.Execute("index", ctx, r, w)
}

func job(w http.ResponseWriter, r *http.Request) {
	ctx := defaultCtx()

	matches := jobIdRegex.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		writeInteralServerError(w, r, fmt.Sprintf("Unable to parse job id in path: %s", r.URL.Path))
		return
	}

	// fmt.Printf("Checking for stream job from URL: %s\n", r.URL.Path)
	if r.URL.Path == "/jobs/" + matches[1] + "/stream" {
		streamJob(w, r)
		return
	}

	renderer.Execute("job", ctx, r, w)
}

func streamJob(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("streamJob executing\n")
	matches := jobIdRegex.FindStringSubmatch(r.URL.Path)
	if matches == nil {
		writeInteralServerError(w, r, "Unable to parse job id")
		return
	}
	jobStr := "job:" + matches[1]

	if r.URL.Path != "/jobs/" + matches[1] + "/stream" {
		writeNotFound(w, r)
		return
	}
	
	outConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error when upgrading to web socket: %s", err)
		writeInteralServerError(w, r, "unable to upgrade to websocket protocol")
		return
	}
	defer outConn.Close()

	cli, err := hub.Subscribe(jobStr)
	if err != nil {
		fmt.Printf("Error when subscribing: %s\n", err)
		return
	}

	for {
		cli.ClientPing()
		if m, ok := <- cli.MsgCh; ok {
			
			err := outConn.WriteMessage(websocket.TextMessage, []byte(m))
			if err != nil {
				fmt.Printf("Unable to write message: %s\n", err)
				// this is how we detect that client closed at this time lol.
				cli.Close()
				removeCmd := pkg.HubCommand{CmdType: pkg.HubCmdRemoveSubscriber, Subscription: jobStr, SubscriberId: cli.Id}
				select {
				case hub.CommandCh <- removeCmd:
				default:
				}
			}
		} else {
			// Subscription was closed
			break
		}			
	}
}

func runJob(jobId int) {
	jobStr := "job:" + strconv.Itoa(jobId)

	_, err := hub.CreateSubscription(jobStr)
	if err != nil {
		fmt.Printf("Unable to create subscription: %s\n", err)
		return
	}

	fmt.Printf("Created subscription: %s\n", jobStr)

	pause, err := time.ParseDuration("1s")
	if err != nil {
		log.Fatalf("Unable to parse duration: %s\n", err)
	}

	time.Sleep(pause)
	hub.PublishTo(jobStr, "Hello 1")
	
	time.Sleep(pause)
	hub.PublishTo(jobStr, "Hello 2")
	
	time.Sleep(pause)
	hub.PublishTo(jobStr, "Hello 3")
	
	time.Sleep(pause)
	hub.PublishTo(jobStr, "Hello 4")

	hub.RemoveSubscription(jobStr)
}

func createJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeInteralServerError(w, r, "Method not supported at this URL")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeInteralServerError(w, r, "Unable to parse form data")
		return
	}

	jobId, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		writeInteralServerError(w, r, "Unable to parse job id")
		return
	}

	id := strconv.Itoa(jobId)
	if sub := hub.GetSubscription("job:" + id); sub != nil {
		http.Redirect(w, r, r.URL.Host + "/jobs/" + id, 302)
		return
	}

	go runJob(jobId)

	http.Redirect(w, r, r.URL.Host + "/jobs/" + id, 302)
}

func main() {
	fmt.Printf("Starting web server...\n")

	hub.Init()
	go hub.Listen()
	
	renderer = render.New(&render.Config{
		ViewPaths:     []string{ "web_app_views" },
		DefaultLayout: "application",
		FuncMapMaker:  nil,
	})

	assetsFs := http.FileServer(http.Dir("./web_assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", assetsFs))
	
	http.Handle("/", http.HandlerFunc(root))
	http.Handle("/jobs", http.HandlerFunc(createJob))
	http.Handle("/jobs/", http.HandlerFunc(job))

	err := http.ListenAndServe("localhost:8081", nil)
	if err != nil {
		log.Fatalf("Error on ListenAndServe: %s\n", err)
	}
}
