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

	"github.com/avct/uasurfer"

	"./database"

	"encoding/hex"

	"encoding/json"

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

// ActionEvent is to represent POST data
type ActionEvent struct {
	Fpt  string
	Vars map[string]string
}

func handler(w http.ResponseWriter, r *http.Request) {

	tags := make(map[string]string)

	var fpt string

	// Get variables from the request
	// TODO: set a limit on arguments

	// GET request
	if r.Method == "GET" {
		for key, vals := range r.URL.Query() {
			log.Printf("%s: %s\n", key, vals[0])

			if key != "fpt" {
				tags[key] = vals[0]
			} else {
				fpt = vals[0]
			}
		}
	} else if r.Method == "POST" {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			r.Response.StatusCode = 400
			fmt.Fprint(w, "Requests must be JSON")
			return
		}

		var postData ActionEvent

		decoder := json.NewDecoder(r.Body)

		err := decoder.Decode(&postData)

		if err != nil {
			r.Response.StatusCode = 500
			log.Println(err)
			return
		}

		fpt = postData.Fpt

		for key, val := range postData.Vars {
			tags[key] = val
		}
	} else {
		r.Response.StatusCode = 405
		return
	}

	ua := r.Header.Get("User-Agent")

	if ua != "" {
		parsedUa := uasurfer.Parse(ua)

		tags["browser"] = parsedUa.Browser.Name.String()[7:] // Remove "Browser" prefix added by uasurfer
		tags["browser_ver"] = VersionString(parsedUa.Browser.Version)

		if parsedUa.Browser.Version.Major != 0 {
			tags["browser_major"] = strconv.Itoa(parsedUa.Browser.Version.Major)
		}

		tags["os"] = parsedUa.OS.Name.String()[2:] // Remove "OS" prefix added by uasurfer

		tags["os_ver"] = VersionString(parsedUa.OS.Version)

		if parsedUa.OS.Version.Major != 0 {
			// OS X versions are weird
			if parsedUa.OS.Name == uasurfer.OSMacOSX {
				tags["os_major"] = strconv.Itoa(parsedUa.OS.Version.Minor)
			} else {
				tags["os_major"] = strconv.Itoa(parsedUa.OS.Version.Major)
			}
		}

		tags["device_type"] = parsedUa.DeviceType.String()[6:] // Remove "Device" prefix added by uasurfer

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

	err = connection.Write(points)

	if err != nil {
		log.Fatal(err)
	}

	// Return a 1px GIF if this is a GET request
	if r.Method == "GET" {
		// TODO: add no-cache header
		w.Header().Add("Content-Type", "image/gif")
		io.Copy(w, bytes.NewReader(OnePixelGIF))
	}
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
