package app

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/gorilla/mux"
)

// WebConfig ...
type WebConfig struct {
	Name        string
	Host        string
	Port        int
	User        string
	Pass        string
	Key         string
	TemplateDir string
	StaticDir   string
}

// Controller web service controller, handles all the work
type Controller interface {
	Close()
}

// Service basic web service
type Service struct {
	Name        string
	Mux         *mux.Router
	Exit        chan error
	Server      *http.Server
	Controllers []Controller
	Home        *url.URL
	Config      *WebConfig
}

// NewWeb instantiates a service with the given name.
func NewWeb(config *WebConfig) *Service {
	// TODO: add default config and/or validate values
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	u, _ := url.Parse(fmt.Sprintf("http://%s", addr))
	app := &Service{
		Name:   config.Name,
		Exit:   make(chan error),
		Home:   u,
		Config: config,
	}

	// Interrupt handler (ctrl-c)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		done := <-signalChan
		app.Exit <- fmt.Errorf("%s", done)
	}()

	// Start Server
	app.Mux = mux.NewRouter()
	app.Mux.HandleFunc("/static/{filename:[a-zA-Z0-9\\.\\-\\_\\/]*}", FileServer)
	app.Mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// favicon

		// brew install imagemagick
		// convert -background none static/db.svg -define icon:auto-resize static/favicon.ico
		if !fileExists("static/favicon.ico") {
			// TODO: spit out generic ico
			fmt.Println("no favicon.ico found")
		}
		w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext("static/favicon.ico")))
		http.ServeFile(w, r, "static/favicon.ico")
	})
	app.Mux.HandleFunc("/health-check", HealthCheck).Methods("GET", "HEAD")
	app.Server = &http.Server{
		Addr:    addr,
		Handler: app.Mux,
	}
	go func() {
		// TODO: add https, stuff...
		fmt.Printf("HTTP server listening on %q\n", addr)
		app.Exit <- app.Server.ListenAndServe()
	}()

	return app
}

// Close http server and any registered controllers
func (s *Service) Close() {
	for _, c := range s.Controllers {
		if c != nil {
			c.Close()
		}
	}

	s.Server.Close()
}

// Register adds controller to list of controllers
func (s *Service) Register(c Controller) {
	s.Controllers = append(s.Controllers, c)
}

// FileServer serves a file with mime type header
func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["filename"]
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(file)))
	http.ServeFile(w, r, "./static/"+file)
}

// HealthCheck return ok
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
