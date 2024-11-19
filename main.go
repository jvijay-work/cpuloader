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
	w.WriteHeader(http.StatusOK)
}

// Handler function for the `/percpu` endpoint.
func loadHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		http.Error(w, "Invalid URL. Expected format: /percpu/{percentage}/{numsecs}", http.StatusBadRequest)
		return
	}

	percentage, err1 := strconv.Atoi(parts[2])
	seconds, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil || percentage < 0 || percentage > 100 || seconds <= 0 {
		http.Error(w, "Invalid percentage or duration", http.StatusBadRequest)
		return
	}

	numCores := runtime.NumCPU()
	runtime.GOMAXPROCS(numCores)

	var wg sync.WaitGroup
	wg.Add(numCores)

	go generateCPULoad(percentage, time.Duration(seconds)*time.Second, numCores, &wg)

	wg.Wait()
	fmt.Fprintf(w, "Completed %d%% CPU load for %d seconds across %d cores\n", percentage, seconds, numCores)
}

// Function to generate CPU load based on the given percentage, duration, and number of cores.
func generateCPULoad(percentage int, duration time.Duration, numCores int, wg *sync.WaitGroup) {
	defer func() {
		for i := 0; i < numCores; i++ {
			wg.Done()
		}
	}()

	endTime := time.Now().Add(duration)
	loadTime := float64(percentage) / 100.0
	restTime := 1.0 - loadTime

	coreLoad := func() {
		defer wg.Done()
		for time.Now().Before(endTime) {
			start := time.Now()
			for time.Since(start).Seconds() < loadTime {
			}
			time.Sleep(time.Duration(restTime * float64(time.Second)))
		}
	}

	for i := 0; i < numCores; i++ {
		wg.Add(1)
		go coreLoad()
	}
}

func main() {
	// Create the server
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/percpu/", loadHandler)

	// Start the server
	go func() {
		fmt.Println("Server is listening on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Failed to start server:", err)
		}
	}()

	// Check for GRACEFULSHUTDOWN environment variable
	gracefulShutdown := strings.ToLower(os.Getenv("GRACEFULSHUTDOWN")) == "true"

	if gracefulShutdown {
		// Set up signal handling and graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		<-sigChan
		fmt.Println("\nShutdown signal received")

		// Use SHUTTIMEOUT environment variable to set the shutdown timeout
		timeout := 10 * time.Second // Default timeout
		if shutTimeoutStr := os.Getenv("SHUTTIMEOUT"); shutTimeoutStr != "" {
			if shutTimeout, err := strconv.Atoi(shutTimeoutStr); err == nil && shutTimeout > 0 {
				timeout = time.Duration(shutTimeout) * time.Second
			} else {
				fmt.Println("Invalid SHUTTIMEOUT value, using default 10 seconds")
			}
		}

		// Simulate background tasks during shutdown
		fmt.Printf("Simulating ongoing tasks for up to %v...\n", timeout)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Add an artificial delay to simulate shutdown tasks
		go func() {
			select {
			case <-ctx.Done():
				fmt.Println("Context deadline reached. Tasks stopped.")
			case <-time.After(timeout):
				fmt.Println("Simulated tasks completed.")
			}
		}()
		// Stop accepting new connections
		server.SetKeepAlivesEnabled(false)
		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			fmt.Println("Error shutting down server:", err)
		} else {
			fmt.Println("Server shut down gracefully")
		}
	} else {
		// Block indefinitely when GRACEFULSHUTDOWN is not enabled
		select {}
	}
}
