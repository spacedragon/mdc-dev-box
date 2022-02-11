package main

//go:generate go run main.go

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type Data struct {
	Duration       int       `json:"duration"`
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
	mdcData := loadFile("./mdc.log")

	fmt.Println("%v", mdcData[0])

	// graph := chart.Chart{
	// 	XAxis: chart.XAxis{
	// 		TickPosition: chart.TickPositionBetweenTicks,
	// 		ValueFormatter: func(v interface{}) string {
	// 			typed := v.(float64)
	// 			typedDate := chart.TimeFromFloat64(typed)
	// 			return fmt.Sprintf("%d-%d\n%d", typedDate.Month(), typedDate.Day(), typedDate.Year())
	// 		},
	// 	},
	// 	Series: []chart.Series{
	// 		chart.ContinuousSeries{
	// 			XValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
	// 			YValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
	// 		},
	// 		chart.ContinuousSeries{
	// 			YAxis:   chart.YAxisSecondary,
	// 			XValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
	// 			YValues: []float64{50.0, 40.0, 30.0, 20.0, 10.0},
	// 		},
	// 	},
	// }

	// f, _ := os.Create("output.png")
	// defer f.Close()
	// graph.Render(chart.PNG, f)
}
