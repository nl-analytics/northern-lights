package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	"image/gif"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/avct/uasurfer"

	"./database"

	"encoding/hex"

	influx "github.com/influxdata/influxdb/client/v2"
)

// VersionString converts a uasurfer.Version into a string
func VersionString(v uasurfer.Version) string {
	if v.Major == 0 {
		return ""
	}

	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}

var b bytes.Buffer
var err = gif.Encode(&b, image.NewAlpha(image.Rect(0, 0, 1, 1)), nil)

// OnePixelGIF - The data for a one pixel transparent GIF
var OnePixelGIF = b.Bytes()

func handler(w http.ResponseWriter, r *http.Request) {

	tags := make(map[string]string)

	var fpt string

	// TODO: set a limit on arguments
	for key, vals := range r.URL.Query() {
		log.Printf("%s: %s\n", key, vals[0])

		if key != "fpt" {
			tags[key] = vals[0]
		} else {
			fpt = vals[0]
		}
	}

	ua := r.Header.Get("User-Agent")

	if ua != "" {
		parsedUa := uasurfer.Parse(ua)

		tags["browser"] = parsedUa.Browser.Name.String()
		tags["browser_ver"] = VersionString(parsedUa.Browser.Version)

		if parsedUa.Browser.Version.Major != 0 {
			tags["browser_major"] = strconv.Itoa(parsedUa.Browser.Version.Major)
		}

		tags["os"] = parsedUa.OS.Name.String()

		tags["os_ver"] = VersionString(parsedUa.OS.Version)

		if parsedUa.OS.Version.Major != 0 {
			// OS X versions are weird
			if parsedUa.OS.Name == uasurfer.OSMacOSX {
				tags["os_major"] = strconv.Itoa(parsedUa.OS.Version.Minor)
			} else {
				tags["os_major"] = strconv.Itoa(parsedUa.OS.Version.Major)
			}
		}

		tags["device_type"] = parsedUa.DeviceType.String()

		log.Printf("%v\n", parsedUa.Browser.Name)
	}

	// TODO: utilized X-Forwarded-For

	// Remove port from IP
	ip := r.RemoteAddr[0:strings.LastIndex(r.RemoteAddr, ":")]

	// Remove square brackets from IPv6
	if ip[0] == '[' {
		ip = ip[1:(len(ip) - 1)]
	}

	tags["ip"] = ip

	// break URL into components
	if tags["url"] != "" {
		pageURL, err := url.Parse(tags["url"])
		if err == nil {
			tags["host"] = pageURL.Host
			tags["scheme"] = pageURL.Scheme
			tags["path"] = pageURL.Path
		}
	}

	connection := database.Connect("test", "test", "http://xenial.dev:8086")
	points, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		Database: "test",
	})

	if err != nil {
		log.Fatal(err)
	}

	pt, _ := influx.NewPoint("hello", tags, map[string]interface{}{
		"fpt": fpt,
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

func randomFingerprint(w http.ResponseWriter, r *http.Request) {
	bytes := make([]byte, 8)

	_, err := rand.Read(bytes)

	if err != nil {
		r.Response.StatusCode = 500
	} else {
		fmt.Fprintf(w, "%s", hex.EncodeToString(bytes))
	}
}

func main() {
	http.HandleFunc("/aurora", handler)
	http.HandleFunc("/fp", randomFingerprint)
	http.ListenAndServe(":3030", nil)
}
