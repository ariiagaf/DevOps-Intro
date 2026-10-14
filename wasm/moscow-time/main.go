
package main

import (
	"fmt"
	"net/http"
	"time"

	spinhttp "github.com/spinframework/spin-go-sdk/v2/http"
)

func init() {
	spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		moscow := time.FixedZone("MSK", 3*60*60)
		now := time.Now().In(moscow)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w,
			`{"unix":%d,"iso":%q,"hour_minute":%q,"timezone":"Europe/Moscow"}`,
			time.Now().Unix(),
			now.Format(time.RFC3339),
			now.Format("15:04"),
		)
	})
}

func main() {}
