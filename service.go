package app

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
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
func NewWeb(conf *WebConfig) *Service {

	// load config from file
	if conf == nil {
		resourceDir := getResourceDir()
		// setting config and payt is redundant if already set in web service
		viper.SetConfigName("config_" + runtime.GOOS)
		viper.AddConfigPath(resourceDir)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Error reading config file, %s", err)
		}
	}

	// load web service config
	var (
		serverConfig *WebConfig
	)
	{
		if conf == nil {
			viper.UnmarshalKey("server", &serverConfig)
		} else {
			serverConfig = conf
		}
	}

	// TODO: validate values
	var addr string
	if serverConfig.Host == "" {
		if serverConfig.Port > 0 {
			// port given, but host isn't assume local host
			serverConfig.Host = "127.0.0.1"
			addr = fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)
		}
	} else if serverConfig.Port <= 0 {
		addr = serverConfig.Host
	} else {
		addr = fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port)
	}
	app := &Service{
		Name:   serverConfig.Name,
		Exit:   make(chan error),
		Config: conf,
	}
	if addr != "" {
		app.Home, _ = url.Parse(fmt.Sprintf("http://%s", addr))
	}

	// Interrupt handler (ctrl-c)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		done := <-signalChan
		app.Exit <- fmt.Errorf("%s", done)
	}()

	// Create Router and add default paths
	app.Mux = mux.NewRouter()
	app.Mux.HandleFunc("/static/{filename:[a-zA-Z0-9\\.\\-\\_\\/]*}", FileServer)
	app.Mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
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

	// Start Server
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

// fileExists checks if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
