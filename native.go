package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	// "github.com/jaredwarren/app/ui"

	"github.com/spf13/viper"
	"github.com/zserge/lorca"
	"golang.org/x/crypto/ssh/terminal"
)

// Config ...
type Config struct {
	Web WebConfig
	UI  NativeConfig
}

// NativeConfig ...
type NativeConfig struct {
	Show   bool
	Width  int
	Height int
}

// Closer ...
type Closer interface {
	Close()
}

// App ...
type App struct {
	Name    string
	Service *Service
	CWD     string
	Exit    chan error
	UI      lorca.UI
}

// NewNative instantiates a service with the given name.
func NewNative(conf *Config) *App {
	var (
		resourceDir = "."
		// storeDir = "."
	)
	{
		// Try to find the resource directory
		// the problem is when running as a native app os.Getwd is ~/ and
		// and when running from commandline (go run) os.Executable is random garbage
		// Good enough for now
		if !terminal.IsTerminal(int(os.Stdout.Fd())) {
			cwd, _ := os.Executable()
			resourceDir = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "Resources")
			// for now because I can't figure out why "Application Support" dir doesn't work
			// storeDir = filepath.Join(filepath.Dir(filepath.Dir(cwd)), "Resources")
			// storeDir = "~/Library/Application Support/invoice"
			// os.Mkdir(storeDir, os.ModePerm)
			os.Chdir(resourceDir)
		} else {
			resourceDir, _ = os.Getwd()
		}
	}

	// TODO: setup logging

	// load config from file
	if conf == nil {
		// setting config and payt is redundant if already set in web service
		viper.SetConfigName("config_" + runtime.GOOS)
		viper.AddConfigPath(resourceDir)
		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Error reading config file, %s", err)
		}
	}

	// load web service config
	var (
		serverConfig WebConfig
	)
	{
		if conf == nil {
			viper.UnmarshalKey("server", &serverConfig)
		} else {
			serverConfig = conf.Web
		}
	}

	var napp = &App{
		CWD:  resourceDir,
		Exit: make(chan error),
	}

	// Setup web app / service
	var (
		wapp *Service
	)
	{
		wapp = NewWeb(&serverConfig)
		go func() {
			done := <-wapp.Exit
			napp.Exit <- fmt.Errorf("%s", done)
		}()
		napp.Service = wapp
	}

	// Interrupt handler (ctrl-c)
	var (
		signalChan = make(chan os.Signal, 1)
	)
	{
		signal.Notify(signalChan, os.Interrupt)
		go func() {
			done := <-signalChan
			napp.Exit <- fmt.Errorf("%s", done)
		}()
	}

	// Setup UI
	var (
		uiConfig NativeConfig
		ui       lorca.UI
		err      error
	)
	{
		if conf == nil {
			viper.UnmarshalKey("ui", &uiConfig)
		} else {
			uiConfig = conf.UI
		}

		// TODO: add other args from config?
		uiArgs := []string{}

		// TODO: make way to update width and height if window is resized, remember size, position?
		width := 500
		if uiConfig.Width > 0 {
			width = uiConfig.Width
		}
		height := 500
		if uiConfig.Height > 0 {
			height = uiConfig.Height
		}

		ui, err = lorca.New("", "", width, height, uiArgs...)
		if err != nil {
			log.Fatal(err)
		}
		napp.UI = ui
	}

	return napp
}

// Run ...
func (a *App) Run() {
	// TODO: might need fall back here, if home is blank...
	a.UI.Load(a.Service.Home.String())
	go func() {
		<-a.UI.Done()
		a.Exit <- fmt.Errorf("UI Closed")
	}()
}

// Close ...
func (a *App) Close() {
	a.Service.Close()
	if a.UI != nil {
		a.UI.Close()
	}
}
