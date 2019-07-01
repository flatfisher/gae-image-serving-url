package main

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"gopkg.in/gographics/imagick.v2/imagick"
)

func main() {
	http.HandleFunc("/", indexHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func getImage() ([]byte, error) {
	ctx := context.Background()
	projectID := ""
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	bucketName := "flatfish"
	bucket := client.Bucket(bucketName)
	bucket.UserProject(projectID)

	obj := bucket.Object("flatfisher.jpg")
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	buff := new(bytes.Buffer)
	if _, err = buff.ReadFrom(r); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// TODO: image と resize 範囲を指定できるようにする
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	b, err := getImage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imgMagic := imagick.NewMagickWand()
	defer imgMagic.Destroy()
	if err := imgMagic.ReadImageBlob(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = imgMagic.ResizeImage(100, 100, imagick.FILTER_POINT, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	blob := imgMagic.GetImageBlob()
	img, err := jpeg.Decode(bytes.NewReader(blob))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	jpeg.Encode(w, img, nil)
}
