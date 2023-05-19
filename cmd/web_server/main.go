package main

import (
	"fmt"
	"log"
	"pkg"
	"os"
	"html/template"
	"strings"
	"net/http"
	"github.com/qor/render"
	"github.com/gorilla/websocket"
)

var renderer *render.Render;
var upgrader = websocket.Upgrader{}

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

func writeInteralServerError(w http.ResponseWriter, msg string) {
	fmt.Printf("Internal Server Error: %s\n", msg)
	writeError(w, msg, http.StatusInternalServerError)
}

func root(w http.ResponseWriter, req *http.Request) {
	ctx := defaultCtx()
	renderer.Execute("index", ctx, req, w)
}

var hub := Hub{}

func webby(w http.ResponseWriter, req *http.Request) {
	outConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Printf("Error when upgrading webby: %s", err)
		writeInteralServerError(w, "unable to upgrade to websocket protocol")
		return
	}
	defer outConn.Close()

	msgChBox, err := hub.subscribe("job:1")

	for {
		if msgChBox.Done() {
			break
		}

		msg := msgChBox.Read()
		outConn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

func monitor() {
	for {
		// This can be replaced with a first wait on the name of the subscription
		// that has published something for the hub to publish.
		for _, name := range hub.subscriptionNames() {
			subscription = hub.GetSubscription(name)

			message <- subscription.Read()
			for subscriber := range subscription.subscribers() {
				outCh = subscriber.ch()
				outCh <- message
			}
		}
	}
}

func launchTask() {
	outCh := hub.createSubscription("job:1")

	pause, err := time.ParseDuration("2s")
	if err != nil {
		log.Fatalf("Unable to parse duration: %s\n", err)
	}

	time.Sleep(pause)
	outCh <- "Hello 1"
	
	time.Sleep(pause)
	outCh <- "Hello 2"
	
	time.Sleep(pause)
	outCh <- "Hello 3"
	
	time.Sleep(pause)
	outCh <- "Hello 4"
	
}

func start_webby(w http.ResponseWriter, req *http.Request) {
	ctx := defaultCtx()

	go launchTask()
	
	renderer.Execute("start_webby", ctx, req, w)
}

func main() {
	fmt.Printf("Starting web server...\n")

	hub.Init()
	
	renderer = render.New(&render.Config{
		ViewPaths:     []string{ "web_app_views" },
		DefaultLayout: "application",
		FuncMapMaker:  nil,
	})

	assetsFs := http.FileServer(http.Dir("./web_assets"))
	http.Handle("/assets/", http.StripPrefix("/assets", assetsFs))
	
	http.Handle("/", http.HandlerFunc(root))
	http.Handle("/webby", http.HandlerFunc(webby))
	http.Handle("/start_webby", http.HandlerFunc(start_webby))

	err := http.ListenAndServe("localhost:8081", nil)
	if err != nil {
		log.Fatalf("Error on ListenAndServe: %s\n", err)
	}
}
