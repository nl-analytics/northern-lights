package main

import (
	"bytes"
	"image"
	"image/gif"
	"io"
	"log"
	"net/http"
)

var b bytes.Buffer
var err = gif.Encode(&b, image.NewAlpha(image.Rect(0, 0, 1, 1)), nil)

// OnePixelGIF - The data for a one pixel transparent GIF
var OnePixelGIF = b.Bytes()

func handler(w http.ResponseWriter, r *http.Request) {
	for key, vals := range r.URL.Query() {
		log.Printf("%s: %s\n", key, vals[0])
	}

	w.Header().Add("Content-Type", "image/gif")
	io.Copy(w, bytes.NewReader(OnePixelGIF))
}

func main() {
	http.HandleFunc("/aurora", handler)
	http.ListenAndServe(":3030", nil)
}
