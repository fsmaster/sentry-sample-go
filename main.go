// go run . error   -> one error from the command line
// go run . serve   -> POST /trigger on :8080 (dashboard)
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var errNoStock = errors.New("not enough stock")

func reserve(sku string, qty int) error {
	stock := map[string]int{"SKU-1": 3}
	if stock[sku] < qty {
		return fmt.Errorf("reserve %s x%d: %w", sku, qty, errNoStock)
	}
	return nil
}

func divide(a, b int) int { return a / b } // panics when b == 0

func generate(kind, run string, fresh bool, dsn string) (id string) {
	hub := sentry.CurrentHub().Clone()
	if dsn != "" {
		// dashboard "Use my DSN": a separate client, only this event goes there
		client, err := sentry.NewClient(sentry.ClientOptions{Dsn: dsn, Environment: "playground", AttachStacktrace: true})
		if err != nil {
			return ""
		}
		hub = sentry.NewHub(client, sentry.NewScope())
		defer client.Flush(5 * time.Second)
	}
	hub.Scope().SetTag("run_id", run)
	if fresh {
		hub.Scope().SetFingerprint([]string{"{{ default }}", run})
	}
	if kind == "panic" {
		defer func() {
			if r := recover(); r != nil {
				id = string(*hub.Recover(r))
			}
		}()
		divide(10, 0)
	}
	err := reserve("SKU-"+run, 5)
	return string(*hub.CaptureException(err))
}

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		Environment:      env("SENTRY_ENVIRONMENT", "local"),
		Release:          env("RELEASE", "sample-go@dev"),
		AttachStacktrace: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sentry.Flush(5 * time.Second)

	cmd := "error"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd != "serve" {
		fmt.Println("event id:", generate(cmd, time.Now().Format("150405"), true, ""))
		return
	}
	http.HandleFunc("POST /trigger", func(w http.ResponseWriter, r *http.Request) {
		body := struct {
			Run   string `json:"run"`
			Kind  string `json:"kind"`
			Fresh *bool  `json:"fresh"`
			Dsn   string `json:"dsn"`
		}{Run: "manual", Kind: "error"}
		json.NewDecoder(r.Body).Decode(&body)
		id := generate(body.Kind, body.Run, body.Fresh == nil || *body.Fresh, body.Dsn)
		sentry.Flush(5 * time.Second)
		json.NewEncoder(w).Encode(map[string]string{"event_id": id})
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
