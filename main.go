package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mrityunjaygr8/todos/templates"
	"github.com/mrityunjaygr8/todos/todos"
)

const IMAGE_NAME = "images/1200.jpg"
const IMAGE_TIME_FILE = "images/time"
const IMAGE_URL = "https://picsum.photos/1200"

func readTimeFile() (time.Time, error) {
	f, err := os.ReadFile(IMAGE_TIME_FILE)
	if err != nil {
		return time.Time{}, err
	}
	if string(f) == "" {
		return time.Time{}, fmt.Errorf("time file is blank")
	}
	ts, err := strconv.ParseInt(string(f), 10, 64)
	if err != nil {
		log.Fatal("Error parsing timestamp", err)
	}
	return time.Unix(ts, 0), nil
}

func refreshTime(lastRefreshTime *time.Time) error {
	log.Println("refreshTime called")
	now := time.Now().UTC()
	file, err := os.OpenFile(IMAGE_TIME_FILE, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	defer file.Close()

	// Write some text to the file
	if _, err := file.WriteString(fmt.Sprintf("%d", now.Unix())); err != nil {
		fmt.Println("Error writing to file:", err)
	}
	*lastRefreshTime = now
	return nil
}

func downloadImage(lastRefreshTime *time.Time) error {
	log.Println("downloadImage has been called")
	resp, err := http.Get(IMAGE_URL)
	if err != nil {
		return fmt.Errorf("error fetching image: %v", err)
	}
	defer resp.Body.Close()
	// Create a new file
	file, err := os.Create(IMAGE_NAME)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving image to file: %v", err)
	}

	return refreshTime(lastRefreshTime)
}

func refreshImage(lastRefreshTime *time.Time) error {
	now := time.Now().UTC()
	hourAgo := now.Add(-1 * time.Hour)
	log.Println("LRT: ", lastRefreshTime, "NOW: ", now, "LAT:", hourAgo)
	if lastRefreshTime.IsZero() || hourAgo.After(*lastRefreshTime) {
		err := downloadImage(lastRefreshTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	log.Println("got /ping request")
	io.WriteString(w, "pong\r\n")
}

func todoHomeHandler(todos []todos.Todo, lastRefreshTime *time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("got / request")
		err := refreshImage(lastRefreshTime)
		if err != nil {
			log.Fatal("An error occurred while refreshing image", err)
		}
		hello := templates.Hello(IMAGE_NAME, todos)
		hello.Render(context.Background(), w)

	}
}
func noCacheFileServer(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers to disable caching
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Cache-Control", "post-check=0, pre-check=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		// Serve the file
		fs.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT has not been provided")
	}

	lrt, err := readTimeFile()
	if err != nil {
		if os.IsNotExist(err) {
			refreshImage(&lrt)
		} else {
			log.Fatal("error reading time file", err)
		}
	}

	todos := []todos.Todo{
		"First", "Second", "Third",
	}

	log.Println("Server started in port", port)
	fs := noCacheFileServer(http.Dir("./images/"))
	http.Handle("/images/", http.StripPrefix("/images/", fs))
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/", todoHomeHandler(todos, &lrt))
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal("whoops")
	}
}
