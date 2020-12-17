package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var folder string = "chunks"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Reuse previous chunks")
	} else if len(args) < 3 {
		panic("Must provide video file, resolution and time chunk as arguments")
	} else {
		fmt.Println("Remove previous chunks")
		os.Remove(folder)
		os.Mkdir(folder, os.ModePerm)
		go func() {
			start := time.Now()
			fmt.Println("Chunk video file")
			err := chunk(args[0], args[1], args[2])
			if err != nil {
				panic(err)
			}

			elapsed := time.Since(start)
			fmt.Println("Video chunk finished", elapsed)
		}()
	}

	http.Handle("/", handlers())
	http.ListenAndServe(":8000", nil)
}

func process() {

}

func handlers() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", indexPage).Methods("GET")
	router.HandleFunc("/media/{mId:[0-9]+}/stream/", streamHandler).Methods("GET")
	router.HandleFunc("/media/{mId:[0-9]+}/stream/{segName:index[0-9]+.ts}", streamHandler).Methods("GET")
	return router
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func streamHandler(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	mId, err := strconv.Atoi(vars["mId"])
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	segName, ok := vars["segName"]
	if !ok {
		mediaBase := getMediaBase(mId)
		m3u8Name := "index.m3u8"
		serveHlsM3u8(response, request, mediaBase, m3u8Name)
	} else {
		mediaBase := getMediaBase(mId)
		serveHlsTs(response, request, mediaBase, segName)
	}
}

func getMediaBase(mId int) string {
	return fmt.Sprintf("%s/%d", folder, mId)
}

func serveHlsM3u8(w http.ResponseWriter, r *http.Request, mediaBase, m3u8Name string) {
	mediaFile := fmt.Sprintf("%s/%s", folder, m3u8Name)
	fmt.Println("mediaFile: ", mediaFile)
	http.ServeFile(w, r, mediaFile)
	w.Header().Set("Content-Type", "application/x-mpegURL")
}

func serveHlsTs(w http.ResponseWriter, r *http.Request, mediaBase, segName string) {
	mediaFile := fmt.Sprintf("%s/%s", folder, segName)
	fmt.Println("mediaFile: ", mediaFile)
	http.ServeFile(w, r, mediaFile)
	w.Header().Set("Content-Type", "video/MP2T")
}

func chunk(file string, resolution string, chunkSize string) error {
	ffmpeg := fmt.Sprintf("ffmpeg -i %s -profile:v baseline -level 3.0 -s %s -start_number 0 -hls_time %s -hls_list_size 0 -f hls %s/index.m3u8", file, resolution, chunkSize, folder)
	fmt.Println("execute cmd", ffmpeg)
	var errOutput bytes.Buffer
	cmd := exec.Command("sh", "-c", ffmpeg)
	cmd.Stderr = &errOutput
	err := cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(errOutput)
		return errors.New("Failed to chunk video file using ffmpeg")
	}

	return nil
}