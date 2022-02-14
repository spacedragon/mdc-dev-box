package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/docker"
	log "github.com/sirupsen/logrus"
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

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	csvFile, err := os.Open("export.csv")
	if err != nil {
		fmt.Println(err)
	}
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

	// findContainers()
	// sampleContainerCpu()

	log.Debug("begin %v", begin)
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

		delay := ts.Sub(begin)
		elapsed := time.Since(now)

		if elapsed < delay {
			wait := delay - elapsed
			time.Sleep(wait)
		}

		go send(&req)
	}
}

var mdc string
var mdcCpu float64
var envoy string
var envoyCpu float64
var model string
var modelCpu float64

func findContainers() {
	dockerlist, err := docker.GetDockerStat()
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range dockerlist {
		if strings.Contains(c.Name, "mdc_1") {
			mdc = c.ContainerID
		}
		if strings.Contains(c.Name, "envoy_1") {
			envoy = c.ContainerID
		}
		if strings.Contains(c.Name, "model_1") {
			model = c.ContainerID
		}
	}
}

func sampleContainerCpu() {
	if len(mdc) > 0 {
		mdcCpu, _ = docker.CgroupCPUUsageDocker(mdc)
	}
	envoyCpu, _ = docker.CgroupCPUUsageDocker(envoy)
	modelCpu, _ = docker.CgroupCPUUsageDocker(model)
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
	_ = doSend("http://localhost:5001", req)
	raw := doSend("http://localhost:5001", req)
	envoyDuration := doSend("http://localhost:10002", req)
	duration := doSend("http://localhost:10001", req)

	// sampleContainerCpu()
	log.WithFields(log.Fields{
		"method":         "Post",
		"statusCode":     200,
		"url":            "",
		"rawDuration":    raw,
		"envoyDuration":  envoyDuration,
		"duration":       duration,
		"serverDuration": req.Time,
		"sent":           req.Sent,
		"recv":           req.Recv,
		"mdcCpu":         mdcCpu,
		"envoyCpu":       envoyCpu,
		"modelCpu":       modelCpu,
	}).Info("Sent request ", req.RequestId)

}

func doSend(target string, req *requestData) int64 {
	u, _ := url.Parse(target)
	q := u.Query()
	q.Set("time", strconv.Itoa(req.Time))
	q.Set("size", strconv.Itoa(req.Sent))
	u.RawQuery = q.Encode()

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
	begin := time.Now()
	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	elapsed := time.Since(begin).Microseconds()
	return elapsed
}
