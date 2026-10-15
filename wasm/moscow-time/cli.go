
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

func main() {
    moscow := time.FixedZone("UTC+3", 3*60*60)
    now := time.Now().In(moscow)

    result := struct {
        Unix       int64  `json:"unix"`
        ISO        string `json:"iso"`
        HourMinute string `json:"hour_minute"`
    }{
        Unix:       now.Unix(),
        ISO:        now.Format(time.RFC3339),
        HourMinute: now.Format("15:04"),
    }

    data, err := json.Marshal(result)
    if err != nil {
        panic(err)
    }

    fmt.Println(string(data))
}

