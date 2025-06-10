package main

import (
	"fmt"
	"github.com/cshum/vipsgen/vips"
	"io"
	"log"
	"net/http"
	"os"
)

func downloadFile(url string, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := downloadFile("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif", "dancing-banana.gif")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	image, err := vips.NewImageFromFile("dancing-banana.gif", &vips.LoadOptions{N: -1})
	if err != nil {
		panic(err)
	}
	defer image.Close()
	if err = image.ExtractAreaMultiPage(30, 40, 50, 70); err != nil {
		panic(err)
	}
	if err = image.Flatten(&vips.FlattenOptions{Background: []float64{0, 255, 255}}); err != nil {
		panic(err)
	}
	err = image.Gifsave("dancing-banana-cropped.gif", nil)
	if err != nil {
		panic(err)
	}
}
