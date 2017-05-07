package main

import (
	"bytes"
	"image"
	"image/gif"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/avct/uasurfer"

	"./database"

	influx "github.com/influxdata/influxdb/client/v2"
)

var b bytes.Buffer
var err = gif.Encode(&b, image.NewAlpha(image.Rect(0, 0, 1, 1)), nil)

// OnePixelGIF - The data for a one pixel transparent GIF
var OnePixelGIF = b.Bytes()

func handler(w http.ResponseWriter, r *http.Request) {

	tags := make(map[string]string)

	for key, vals := range r.URL.Query() {
		log.Printf("%s: %s\n", key, vals[0])
		tags[key] = vals[0]
	}

	ua := r.Header.Get("User-Agent")

	if ua != "" {
		parsedUa := uasurfer.Parse(ua)
		log.Printf("%v\n", parsedUa.Browser.Name)
	}

	connection := database.Connect("test", "test", "http://xenial.dev:8086")
	points, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database: "test",
	})

	if err != nil {
		log.Fatal(err)
	}

	pt, _ := influx.NewPoint("hello", tags, map[string]interface{}{
		"hello": 51.5,
	})

	points.AddPoint(pt)

	log.Println(connection.Ping(time.Second))
	err = connection.Write(points)

	if err != nil {
		log.Fatal(err)
	}

	w.Header().Add("Content-Type", "image/gif")
	io.Copy(w, bytes.NewReader(OnePixelGIF))
}

func main() {
	http.HandleFunc("/aurora", handler)
	http.ListenAndServe(":3030", nil)
}
