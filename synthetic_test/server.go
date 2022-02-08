package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {

	http.HandleFunc("/", handler)
	fmt.Println("Server started at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	duration, ok := query["time"]
	if ok {
		d, _ := strconv.Atoi(duration[0])
		time.Sleep(time.Duration(d) * time.Millisecond)
	}

	size, ok := query["size"]
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
	if ok {
		s, _ := strconv.Atoi(size[0])
		if s >= 2 {
			b := make([]byte, s, s)
			b[0] = '"'
			b[len(b)-1] = '"'
			for j := 1; j < len(b)-1; j++ {
				b[j] = 'k'
			}
			w.Write(b)
		}
	}
	log.Printf("%s %s\n", r.Method, r.URL)
}
