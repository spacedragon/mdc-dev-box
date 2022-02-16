package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	sse "github.com/alexandrevicenzi/go-sse"
)

func main() {
	http.HandleFunc("/score", handler)

	http.HandleFunc("/ready", func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(200) })

	opt := &sse.Options{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "https://localhost:5001",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     "GET, OPTIONS",
			"Access-Control-Allow-Headers":     "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID",
		},
	}
	s := sse.NewServer(opt)
	http.Handle("/events", s)
	defer s.Shutdown()

	id := 1
	go func() {
		for {
			eventType := "ping"
			data := make([]byte, 1024)
			for i := 0; i < 1024; i++ {
				data[i] = 'o'
			}
			s.SendMessage("/events", sse.NewMessage(strconv.Itoa(id), string(data), eventType))
			id++
			time.Sleep(100 * time.Millisecond)
		}
	}()

	fmt.Println("Server started at port 5001")
	log.Fatal(http.ListenAndServe(":5001", nil))
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
		log.Printf("%s", size[0])
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
