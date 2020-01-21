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
	Config      *Config
}

// New instantiates a service with the given name.
func New(config *Config) *Service {
	// TODO: add default config and/or validate values
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	u, _ := url.Parse(fmt.Sprintf("http://%s", addr))
	var app = &Service{
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
