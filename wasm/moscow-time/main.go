package main

import (
    "encoding/json"
    "net/http"
    "time"

    spinhttp "github.com/spinframework/spin-go-sdk/v2/http"
)

func init() {
    spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.Header().Set("Allow", "GET")
            http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
            return
        }

        moscow := time.FixedZone("UTC+3", 3*60*60)
        now := time.Now().In(moscow)

        response := struct {
            Unix       int64  `json:"unix"`
            ISO        string `json:"iso"`
            HourMinute string `json:"hour_minute"`
        }{
            Unix:       now.Unix(),
            ISO:        now.Format(time.RFC3339),
            HourMinute: now.Format("15:04"),
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })
}

func main() {}