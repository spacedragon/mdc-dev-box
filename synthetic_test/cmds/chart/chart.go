package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	chart "github.com/wcharczuk/go-chart/v2"
)

type Data struct {
	Duration       int       `json:"duration"`
	RawDuration    int       `json:"rawDuration"`
	EnvoyDuration  int       `json:"envoyDuration"`
	EnvoyCPU       float64   `json:"envoyCpu"`
	Level          string    `json:"level"`
	MdcCPU         float64   `json:"mdcCpu"`
	Method         string    `json:"method"`
	ModelCPU       float64   `json:"modelCpu"`
	Msg            string    `json:"msg"`
	Recv           int       `json:"recv"`
	Sent           int       `json:"sent"`
	ServerDuration int       `json:"serverDuration"`
	StatusCode     int       `json:"statusCode"`
	Time           time.Time `json:"time"`
	URL            string    `json:"url"`
}

func loadFile(file string) []Data {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 4*1024)
	ret := []Data{}

	line, _, err := r.ReadLine()
	for err == nil {
		d := Data{}
		if err = json.Unmarshal(line, &d); err == nil {
			ret = append(ret, d)
		}

		line, _, err = r.ReadLine()
	}
	return ret
}

func main() {
	mdcData := loadFile("./result.log")

	xValues := make([]time.Time, 0)
	mdc_durations := make([]float64, 0)
	add_durations := make([]float64, 0)
	server_durations := make([]float64, 0)
	envoy_durations := make([]float64, 0)

	for idx, d := range mdcData {

		if idx%200 == 0 && d.StatusCode == 200 {
			xValues = append(xValues, d.Time)
			mdc_durations = append(mdc_durations, float64(d.Duration))
			server_durations = append(server_durations, float64(d.ServerDuration))
			add_durations = append(add_durations, float64(d.Duration-d.ServerDuration))
			envoy_durations = append(envoy_durations, float64(d.EnvoyDuration))
		}
	}

	graph := chart.Chart{
		YAxis: chart.YAxis{
			Name: "Elapsed Millis",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%d ms", int(v.(float64)))
			},
		},
		XAxis: chart.XAxis{
			ValueFormatter: chart.TimeValueFormatterWithFormat("03:04:05"),
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "server",
				YAxis:   chart.YAxisSecondary,
				XValues: xValues,
				YValues: server_durations,
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
					FillColor:   chart.ColorBlue.WithAlpha(100),
				},
			},

			chart.TimeSeries{
				Name:    "without mdc",
				YAxis:   chart.YAxisSecondary,
				XValues: xValues,
				YValues: envoy_durations,
				Style: chart.Style{
					StrokeColor: chart.ColorYellow,
					FillColor:   chart.ColorYellow.WithAlpha(20),
				},
			},
			chart.TimeSeries{
				Name:    "mdc",
				XValues: xValues,
				YValues: mdc_durations,
				Style: chart.Style{
					StrokeColor: chart.ColorCyan,
					FillColor:   chart.ColorCyan.WithAlpha(20),
				},
			},
			chart.TimeSeries{
				Name:    "mdc added latency",
				XValues: xValues,
				YValues: add_durations,
				Style: chart.Style{
					StrokeColor:     chart.ColorRed,
					StrokeDashArray: []float64{5.0, 5.0},
				},
			},
			// chart.TimeSeries{
			// 	Name:    "payload_size",
			// 	XValues: xValues,
			// 	YValues: payload_size,
			// 	Style: chart.Style{
			// 		StrokeColor: chart.ColorBlack,
			// 	},
			// },
		},
	}
	graph.Elements = []chart.Renderable{chart.LegendThin(&graph)}

	f, _ := os.Create("output.png")
	defer f.Close()
	graph.Render(chart.PNG, f)

	sort.Slice(mdcData, func(i, j int) bool {
		return (mdcData[i].Sent + mdcData[i].Recv) < (mdcData[j].Sent + mdcData[j].Recv)
	})

	mdc_cpu := make([]float64, 0)
	envoy_cpu := make([]float64, 0)
	payload_size := make([]float64, 0)

	for idx, d := range mdcData {
		if idx%200 == 0 {

			mdc_cpu = append(mdc_cpu, d.MdcCPU)
			envoy_cpu = append(envoy_cpu, d.EnvoyCPU)
			payload_size = append(payload_size, float64(d.Sent+d.Recv)/1024)
		}
	}

	graph2 := chart.Chart{
		YAxis: chart.YAxis{
			Name: "Cpu",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%d %%", int(v.(float64)))
			},
		},
		XAxis: chart.XAxis{
			Name: "Payload Size",
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%d k", int(v.(float64)))
			},
			//GridLines: releases(),
		},
		Series: []chart.Series{

			chart.ContinuousSeries{
				Name:    "mdc cpu",
				XValues: payload_size,
				YValues: mdc_cpu,
				Style: chart.Style{
					StrokeColor: chart.ColorCyan,
				},
			},
			chart.ContinuousSeries{
				Name:    "envoy cpu",
				XValues: payload_size,
				YValues: envoy_cpu,
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
				},
			},
		},
	}
	graph2.Elements = []chart.Renderable{chart.LegendThin(&graph2)}

	f2, _ := os.Create("size.png")
	defer f.Close()
	graph2.Render(chart.PNG, f2)

	fmt.Printf("Server P50 %.03f \n", calculatePXX(50, mdcData, raw_duration))
	fmt.Printf("Server P90 %.03f \n", calculatePXX(90, mdcData, raw_duration))
	fmt.Printf("Server P99 %.03f \n", calculatePXX(99, mdcData, raw_duration))

	fmt.Printf("\nWithout MDC P50 %.03f \n", calculatePXX(50, mdcData, envoy_duration))
	fmt.Printf("Without MDC P95 %.03f \n", calculatePXX(95, mdcData, envoy_duration))
	fmt.Printf("Without MDC P99 %.03f \n", calculatePXX(99, mdcData, envoy_duration))

	fmt.Printf("\nMDC P50 %.03f \n", calculatePXX(50, mdcData, mdc_duration))
	fmt.Printf("MDC P95 %.03f \n", calculatePXX(95, mdcData, mdc_duration))
	fmt.Printf("MDC P99 %.03f \n", calculatePXX(99, mdcData, mdc_duration))

	fmt.Printf("\nEnvoy Added P50 %.03f \n", calculatePXX(50, mdcData, envoy_added))
	fmt.Printf("Envoy Added P95 %.03f \n", calculatePXX(95, mdcData, envoy_added))
	fmt.Printf("Envoy Added P99 %.03f \n", calculatePXX(99, mdcData, envoy_added))

	fmt.Printf("\nEnvoy+MDC Added P50 %.03f \n", calculatePXX(50, mdcData, mdc_added))
	fmt.Printf("Envoy+MDC Added P95 %.03f \n", calculatePXX(95, mdcData, mdc_added))
	fmt.Printf("Envoy+MDC Added P99 %.03f \n", calculatePXX(99, mdcData, mdc_added))

	fmt.Printf("\nSize P50 %.1fk \n", calculatePXX(50, mdcData, payloadSize))
	fmt.Printf("Size P90 %.1fk \n", calculatePXX(90, mdcData, payloadSize))
	fmt.Printf("Size P99 %.1fk \n", calculatePXX(99, mdcData, payloadSize))
}

func calculatePXX(p int, data []Data, f func(*Data) float64) float64 {
	sort.Slice(data, func(i, j int) bool {
		return f(&data[i]) < f(&data[j])
	})

	idx := len(data) * p / 100
	return f(&data[idx])
}

func raw_duration(d *Data) float64 {
	return float64(d.RawDuration) / 1000
}

func mdc_duration(d *Data) float64 {
	return float64(d.Duration) / 1000
}

func envoy_duration(d *Data) float64 {
	return float64(d.EnvoyDuration) / 1000
}

func payloadSize(d *Data) float64 {
	return float64(d.Sent+d.Recv) / 1024
}

func envoy_added(d *Data) float64 {
	return float64(d.EnvoyDuration-d.RawDuration) / 1000
}

func mdc_added(d *Data) float64 {
	return float64(d.Duration-d.RawDuration) / 1000
}
