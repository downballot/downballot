package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/downballot/downballot/internal/api"
	"github.com/downballot/downballot/internal/appconfig"
	"github.com/downballot/downballot/internal/application"
	"github.com/downballot/downballot/internal/database"
	"github.com/downballot/downballot/internal/httpextra"
	"github.com/downballot/downballot/internal/migrator"
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

			if logLevel == logrus.DebugLevel {
				slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
			}
		} else {
			slog.WarnContext(ctx, fmt.Sprintf("Unknown log level: %q", value))
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
		slog.InfoContext(ctx, fmt.Sprintf("Slack channel: %s", slackChannel))
		slog.InfoContext(ctx, fmt.Sprintf("Slack level: %s", slackLevel))
		if slackToken == "" {
			slog.InfoContext(ctx, "Slack token: n/a")
		} else {
			slog.InfoContext(ctx, "Slack token: ********")
		}
		debugSlack := false
		if value := os.Getenv("SLACK_DEBUG"); value != "" {
			var err error
			debugSlack, err = strconv.ParseBool(value)
			if err != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Could not parse value %q: %v", value, err))
				os.Exit(1)
			}
		}
		slog.InfoContext(ctx, fmt.Sprintf("Debug slack: %t", debugSlack))
		if slackToken != "" && slackChannel != "" {
			// Parse the slack log level.
			minimumSlackLogLevel, err := logrus.ParseLevel(slackLevel)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Unknown log level: %q", slackLevel))
				minimumSlackLogLevel = logrus.ErrorLevel // Default to the error level.
			}

			slackClient := slack.New(slackToken, slack.OptionDebug(debugSlack))
			hook := slackhook.New(slackClient, slackChannel, minimumSlackLogLevel)
			logrus.AddHook(hook)

			slog.InfoContext(ctx, "Slack hook has been registered.")

			/* Re-enable these to verify the Slack hook is working appropriately.
			slog.DebugContext(ctx, "Debug")
			slog.InfoContext(ctx, "Info")
			slog.WarnContext(ctx, "Warn")
			slog.ErrorContext("Error")
			os.Exit(0)
			//*/
		}
	}

	// Print the Go runtime information.
	slog.InfoContext(ctx, fmt.Sprintf("Go runtime: Version: %s", runtime.Version()))
	slog.InfoContext(ctx, fmt.Sprintf("Go runtime: NumCPU: %d", runtime.NumCPU()))

	portString := os.Getenv("PORT")
	if portString == "" {
		portString = "8888"
	}
	staticDirectory := os.Getenv("STATIC_DIR")
	if staticDirectory == "" {
		staticDirectory = "static"
	}

	slog.InfoContext(ctx, fmt.Sprintf("Port: %s", portString))
	slog.InfoContext(ctx, fmt.Sprintf("Static directory: %s", staticDirectory))

	profile := false
	if value := os.Getenv("PROFILE"); value != "" {
		v, err := strconv.ParseBool(value)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Could not parse PROFILE value: %v", err))
		} else {
			profile = v
		}
	}

	slog.InfoContext(ctx, fmt.Sprintf("Port: %s", portString))
	slog.InfoContext(ctx, fmt.Sprintf("Static directory: %s", staticDirectory))

	{
		dir, err := os.Getwd()
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Current working directory: Error: %v", err))
		} else {
			slog.InfoContext(ctx, fmt.Sprintf("Current working directory: %s", dir))
		}

		slog.InfoContext(ctx, "Files in the current directory:")
		var files []string
		fileInfo, err := os.ReadDir(".")
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Error listing the files: %v", err))
		} else {
			for _, file := range fileInfo {
				name := file.Name()
				if file.IsDir() {
					name += "/"
				}
				files = append(files, name)
			}
			slices.Sort(files)
			for _, file := range files {
				slog.InfoContext(ctx, fmt.Sprintf("* %s", file))
			}
		}
	}
	{
		slog.InfoContext(ctx, "Files in the static directory:")
		var files []string
		fileInfo, err := os.ReadDir(staticDirectory)
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Error listing the files: %v", err))
		} else {
			for _, file := range fileInfo {
				name := file.Name()
				if file.IsDir() {
					name += "/"
				}
				files = append(files, name)
			}
			slices.Sort(files)
			for _, file := range files {
				slog.InfoContext(ctx, fmt.Sprintf("* %s", file))
			}
		}
	}

	var config appconfig.Config
	if _, err := os.Stat("config.json"); err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("Could not find config.json: %v", err))
	} else {
		contents, err := os.ReadFile("config.json")
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Could not read config.json: %v", err))
			os.Exit(1)
		}
		err = json.Unmarshal(contents, &config)
		if err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Could not parse config.json: %v", err))
			os.Exit(1)
		}
	}

	db, err := database.New(ctx, config.DatabaseDriver, config.DatabaseString)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Could not connect to database: %v", err))
		os.Exit(1)
	}

	err = migrator.Migrate(db)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Could not migrate database: %v", err))
		os.Exit(1)
	}

	app := application.New(db)

	apiInstance := api.New()
	apiInstance.App = app
	apiInstance.Config = api.Config{
		JWTPrivateKey: config.JWTPrivateKey,
		JWTPublicKey:  config.JWTPublicKey,
		JWTSecret:     config.JWTSecret,
		MasterToken:   config.MasterToken,
	}

	apiContainer := apiInstance.Container(ctx)

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

	slog.InfoContext(ctx, fmt.Sprintf("Listening on: %s", httpServer.Addr))
	err = httpServer.ListenAndServe()
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error: %v", err))
	}
}
