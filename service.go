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
	"github.com/gorilla/websocket"
)

// Closer ...
type Closer interface {
	Close()
}

// Service ...
type Service struct {
	Name        string
	Mux         *mux.Router
	Exit        chan error
	Server      *http.Server
	Controllers []Closer
	Home        *url.URL
	Config      *Config
}

// New instantiates a service with the given name.
func New(config *Config) *Service {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	u, _ := url.Parse(fmt.Sprintf("http://%s", addr))
	var service = &Service{
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
		// service.Close() // close is called later, not sure where it's better to call
		service.Exit <- fmt.Errorf("%s", done)
	}()

	// Start Server
	service.Mux = mux.NewRouter()
	service.Mux.HandleFunc("/static/{filename:[a-zA-Z0-9\\.\\-\\_\\/]*}", FileServer)
	service.Server = &http.Server{
		Addr:    addr,
		Handler: service.Mux,
	}
	go func() {
		// TODO: add https, stuff...
		fmt.Printf("HTTP server listening on %q\n", addr)
		service.Exit <- service.Server.ListenAndServe()
	}()

	return service
}

// Close ...
func (s *Service) Close() {
	s.Server.Close()
}

// Register ...
func (s *Service) Register(c Closer) {
	s.Controllers = append(s.Controllers, c)
}

// FileServer serves a file with mime type header
func FileServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["filename"]
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(file)))
	http.ServeFile(w, r, "./static/"+file)
}

// InternalError ...
func InternalError(ws *websocket.Conn, msg string, err error) {
	fmt.Println(msg, err)
	ws.WriteMessage(websocket.TextMessage, []byte("Internal server error."))
}
