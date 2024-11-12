package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
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
	// Create an HTTP server
	server := &http.Server{Addr: ":8080"}

	// Register the root handler for "/"
	http.HandleFunc("/", rootHandler)

	// Register the handler function for the `/percpu` endpoint
	http.HandleFunc("/percpu/", loadHandler)

	// Channel to listen for interrupt or terminate signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Run the server in a goroutine so it doesn’t block
	go func() {
		fmt.Println("Server is listening on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Failed to start server:", err)
		}
	}()

	// Block until we receive a signal
	<-sigChan
	fmt.Println("\nShutdown signal received")

	// Create a context with a timeout for the graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt a graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("Error shutting down server:", err)
	} else {
		fmt.Println("Server shut down gracefully")
	}
}
