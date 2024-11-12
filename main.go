package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Root handler for responding to "/".
func rootHandler(w http.ResponseWriter, _ *http.Request) {
	// Return a 200 OK with an empty body
	w.WriteHeader(http.StatusOK)
}

// Handler function for the `/percpu` endpoint.
func loadHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the URL path segments
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(w, "Invalid URL. Expected format: /percpu/{percentage}/{numsecs}", http.StatusBadRequest)
		return
	}

	// Parse the percentage and duration from the URL path
	percentage, err1 := strconv.Atoi(parts[2])
	seconds, err2 := strconv.Atoi(parts[3])

	if err1 != nil || err2 != nil || percentage < 0 || percentage > 100 || seconds <= 0 {
		http.Error(w, "Invalid percentage or duration", http.StatusBadRequest)
		return
	}

	// Set GOMAXPROCS to the number of available CPU cores
	numCores := runtime.NumCPU()
	runtime.GOMAXPROCS(numCores)

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(numCores)

	// Start generating the CPU load
	go generateCPULoad(percentage, time.Duration(seconds)*time.Second, numCores, &wg)

	// Wait for all CPU load goroutines to complete
	wg.Wait()

	// Send a response indicating the load generation completion
	fmt.Fprintf(w, "Completed %d%% CPU load for %d seconds across %d cores\n", percentage, seconds, numCores)
}

// Function to generate CPU load based on the given percentage, duration, and number of cores.
func generateCPULoad(percentage int, duration time.Duration, numCores int, wg *sync.WaitGroup) {
	defer func() {
		// Mark all as done on completion
		for i := 0; i < numCores; i++ {
			wg.Done()
		}
	}()

	endTime := time.Now().Add(duration)
	loadTime := float64(percentage) / 100.0
	restTime := 1.0 - loadTime

	// Function for each core to generate the CPU load
	coreLoad := func() {
		defer wg.Done() // Signal completion when the goroutine finishes

		for time.Now().Before(endTime) {
			start := time.Now()
			// Busy-loop to simulate CPU load
			for time.Since(start).Seconds() < loadTime {
			}
			// Rest to achieve the specified load percentage
			time.Sleep(time.Duration(restTime * float64(time.Second)))
		}
	}

	// Launch a goroutine for each core
	for i := 0; i < numCores; i++ {
		wg.Add(1) // Add a counter for each new goroutine
		go coreLoad()
	}
}

func main() {
	// Register the root handler for "/"
	http.HandleFunc("/", rootHandler)

	// Register the handler function for the `/percpu` endpoint
	http.HandleFunc("/percpu/", loadHandler)

	// Start the server
	fmt.Println("Server is listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
