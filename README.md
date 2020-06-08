# APP
Super simple base web application. I just got tired of copying the same code to every project. 


## config 
### Config Format
```yml
server:
  name: Host # Name/Title of web service
  host: 127.0.0.1 # TCP address to listen on
  port: 8081 # TCP port to listen on

# Native Config Options
ui:
  show: true
  width: 600
  height: 600
```

### Load
```go
// load config
	viper.SetConfigName("config_" + runtime.GOOS)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	var serverConfig app.Config
	viper.UnmarshalKey("server", &serverConfig)
	fmt.Printf("%+v\n", serverConfig)
```

## Usage
### Web
```go
    conf := &app.WebConfig{
		Host: "127.0.0.1",
		Port: 8084,
	}
	a := app.NewWeb(conf)
	service.Register(a)
	d := <-a.Exit
``` 

### Native
```go
    app := app.NewNative(nil)
    defer app.Close()
    service.Register(app.Service)
    app.Run()
    done := <-app.Exit
    if done != nil {
        fmt.Println("Something Happened, Bye!", done)
    } else {
        fmt.Println("Good Bye!")
    }
``` 