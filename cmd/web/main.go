package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/downballot/downballot/internal/api"
	"github.com/downballot/downballot/internal/appconfig"
	"github.com/downballot/downballot/internal/application"
	"github.com/downballot/downballot/internal/database"
	"github.com/downballot/downballot/internal/httpextra"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/slackhook"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func main() {
	ctx := context.Background()

	var err error

	// Set the log level.
	if value, ok := os.LookupEnv("LOG_LEVEL"); ok {
		logLevel, err := logrus.ParseLevel(value)
		if err == nil {
			logrus.SetLevel(logLevel)
		} else {
			logrus.Warnf("Unknown log level: %q", value)
		}
	}

	// Handle Slack early on, since the results from this section will dictate how
	// logs are treated.
	{
		slackChannel := os.Getenv("SLACK_CHANNEL")
		slackLevel := os.Getenv("SLACK_LEVEL")
		if slackLevel == "" {
			slackLevel = "error"
		}
		slackToken := os.Getenv("SLACK_TOKEN")
		logrus.Infof("Slack channel: %s", slackChannel)
		logrus.Infof("Slack level: %s", slackLevel)
		if slackToken == "" {
			logrus.Infof("Slack token: n/a")
		} else {
			logrus.Infof("Slack token: ********")
		}
		debugSlack := false
		if value := os.Getenv("SLACK_DEBUG"); value != "" {
			var err error
			debugSlack, err = strconv.ParseBool(value)
			if err != nil {
				logrus.Errorf("Could not parse value %q: %v", value, err)
				os.Exit(1)
			}
		}
		logrus.Infof("Debug slack: %t", debugSlack)
		if slackToken != "" && slackChannel != "" {
			// Parse the slack log level.
			minimumSlackLogLevel, err := logrus.ParseLevel(slackLevel)
			if err != nil {
				logrus.Warnf("Unknown log level: %q", slackLevel)
				minimumSlackLogLevel = logrus.ErrorLevel // Default to the error level.
			}

			slackClient := slack.New(slackToken, slack.OptionDebug(debugSlack))
			hook := slackhook.New(slackClient, slackChannel, minimumSlackLogLevel)
			logrus.AddHook(hook)

			logrus.Infof("Slack hook has been registered.")

			/* Re-enable these to verify the Slack hook is working appropriately.
			logrus.Tracef("Trace")
			logrus.Debugf("Debug")
			logrus.Infof("Info")
			logrus.Warnf("Warn")
			logrus.Errorf("Error")
			logrus.Fatalf("Fatal")
			logrus.Panicf("Panic")
			os.Exit(0)
			//*/
		}
	}

	// Print the Go runtime information.
	logrus.Infof("Go runtime: Version: %s", runtime.Version())
	logrus.Infof("Go runtime: NumCPU: %d", runtime.NumCPU())

	portString := os.Getenv("PORT")
	if portString == "" {
		portString = "8888"
	}
	staticDirectory := os.Getenv("STATIC_DIR")
	if staticDirectory == "" {
		staticDirectory = "static"
	}

	logrus.Infof("Port: %s", portString)
	logrus.Infof("Static directory: %s", staticDirectory)

	profile := false
	if value := os.Getenv("PROFILE"); value != "" {
		v, err := strconv.ParseBool(value)
		if err != nil {
			logrus.Warnf("Could not parse PROFILE value: %v", err)
		} else {
			profile = v
		}
	}

	logrus.Infof("Port: %s", portString)
	logrus.Infof("Static directory: %s", staticDirectory)

	{
		dir, err := os.Getwd()
		if err != nil {
			logrus.Warnf("Current working directory: Error: %v", err)
		} else {
			logrus.Infof("Current working directory: %s", dir)
		}

		logrus.Infof("Files in the current directory:")
		var files []string
		fileInfo, err := ioutil.ReadDir(".")
		if err != nil {
			logrus.Warnf("Error listing the files: %v", err)
		} else {
			for _, file := range fileInfo {
				name := file.Name()
				if file.IsDir() {
					name += "/"
				}
				files = append(files, name)
			}
			sort.Strings(files)
			for _, file := range files {
				logrus.Infof("* %s", file)
			}
		}
	}
	{
		logrus.Infof("Files in the static directory:")
		var files []string
		fileInfo, err := ioutil.ReadDir(staticDirectory)
		if err != nil {
			logrus.Warnf("Error listing the files: %v", err)
		} else {
			for _, file := range fileInfo {
				name := file.Name()
				if file.IsDir() {
					name += "/"
				}
				files = append(files, name)
			}
			sort.Strings(files)
			for _, file := range files {
				logrus.Infof("* %s", file)
			}
		}
	}

	var config appconfig.Config
	if _, err := os.Stat("config.json"); err != nil {
		logrus.Warnf("Could not find config.json: %v", err)
	} else {
		contents, err := ioutil.ReadFile("config.json")
		if err != nil {
			logrus.Errorf("Could not read config.json: %v", err)
			os.Exit(1)
		}
		err = json.Unmarshal(contents, &config)
		if err != nil {
			logrus.Errorf("Could not parse config.json: %v", err)
			os.Exit(1)
		}
	}

	app := application.New()
	app.DB, err = database.New(ctx, config.DatabaseDriver, config.DatabaseString)
	if err != nil {
		logrus.Errorf("Could not connect to database: [%T] %v", err, err)
		os.Exit(1)
	}
	{
		err := app.DB.AutoMigrate(
			schema.Organization{},
			schema.User{},
		)
		if err != nil {
			logrus.Errorf("Could not auto-migrate database: [%T] %v", err, err)
			os.Exit(1)
		}
	}

	apiInstance := api.New()
	apiInstance.App = app
	apiInstance.Config = config

	apiContainer := apiInstance.Container()

	myHandler := http.NewServeMux()
	staticHandler := http.StripPrefix("/", http.FileServer(http.Dir(staticDirectory)))

	myHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/", http.StatusFound)
			return
		}

		/*
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
			w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
			w.Header().Set("Expires", "0")                                         // Proxies.
		*/

		// Cache all the static files aligned at the 1-minute boundary.
		expirationTime := time.Now().Truncate(1 * time.Minute).Add(1 * time.Minute)
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%0.0f, must-revalidate", time.Until(expirationTime).Seconds()))
		w.Header().Set("ETag", fmt.Sprintf("W/\"exp_%d\"", expirationTime.Unix())) // The ETag is weak ("W/" prefix) because it'll be the same tag for all encodings.

		// Strip the headers that `http.FileServer` will use that rely on modification time.
		// App Engine sets all of the timestamps to January 1, 1980.
		r.Header.Del("If-Modified-Since")
		r.Header.Del("If-Unmodified-Since")

		staticHandler.ServeHTTP(w, r)
	})
	myHandler.Handle("/api/", apiContainer)

	if profile {
		//runtime.SetCPUProfileRate(50000)
		myHandler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		myHandler.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		myHandler.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		myHandler.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		myHandler.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	}

	var finalHandler http.Handler
	finalHandler = httpextra.LogHandler("web", myHandler)
	httpServer := &http.Server{
		Addr:    ":" + portString,
		Handler: finalHandler,
	}

	logrus.Infof("Listening on: %s", httpServer.Addr)
	err = httpServer.ListenAndServe()
	if err != nil {
		logrus.Errorf("Error: %v", err)
	}
}
