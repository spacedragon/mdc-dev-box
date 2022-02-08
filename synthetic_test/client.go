package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type requestData struct {
	TimeStamp time.Time
	RequestId string
	Code      string
	Time      int
	Recv      int
	Sent      int
}

func main() {

	csvFile, err := os.Open("export.csv")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened CSV file")
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.LazyQuotes = true
	csvLines, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
	}
	layout := "2006-01-02T03:04:05Z"
	begin, err := time.Parse(layout, trimQuotes(csvLines[0][0]))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("begin %v", begin)
	now := time.Now()
	for _, line := range csvLines {
		ts, _ := time.Parse(layout, trimQuotes(line[0]))
		t, _ := strconv.Atoi(line[3])
		r, _ := strconv.Atoi(line[4])
		s, _ := strconv.Atoi(line[5])

		req := requestData{
			RequestId: line[1],
			Code:      line[2],
			Time:      t,
			Recv:      r,
			Sent:      s,
		}
		log.Printf("now %v", ts)

		delay := ts.Sub(begin)
		elapsed := time.Since(now)

		if elapsed < delay {
			wait := delay - elapsed
			log.Printf("wait for %v", wait)
			time.Sleep(wait)
		}

		go send(&req)
	}
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		b := strings.Index(s, "\"") + 1
		e := strings.LastIndex(s, "\"")
		if (b > -1) && e > -1 {
			return s[b:e]
		}
	}
	return s
}

func send(req *requestData) {
	url := fmt.Sprintf("http://localhost:8080/?time=%d&size=%d", req.Time, req.Sent)
	log.Println(url)
	var b []byte
	if req.Recv > 2 {
		b = make([]byte, req.Recv, req.Recv)
		b[0] = '"'
		b[len(b)-1] = '"'
		for j := 1; j < len(b)-1; j++ {
			b[j] = 'o'
		}
	} else {
		b = []byte{}
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}
