package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {

	fmt.Println("Go SpeedTest")

	target := flag.String("target", "", "HTTP remote URL for speed testing")
	concurrent := flag.Int64("concurrent", 4, "Number of parallel downloads")
	duration := flag.Int("duration", 0, "Stop the download after xx seconds")
	progress := flag.Bool("progress", false, "Display real-time progress bar")

	flag.Parse()

	if *target == "" {
		fmt.Println("Target URL is required.")
		os.Exit(1)
	}

	// Get the file size
	resp, err := http.Head(*target)
	if err != nil {
		fmt.Printf("Failed to get file size: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fileSize := resp.ContentLength
	if fileSize <= 0 {
		fmt.Println("Invalid file size.")
		os.Exit(1)
	}

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	start := time.Now()

	// Channel to signal the end of the test
	done := make(chan struct{})

	// Channel to capture interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Ticker to update progress bars every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Function to download a part of the file
	downloadPart := func(part int64, progressCounters []int64) {
		defer wg.Done()
		req, _ := http.NewRequest("GET", *target, nil)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part*fileSize/int64(*concurrent), (part+1)*fileSize/int64(*concurrent)-1))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Failed to download part %d: %v\n", part, err)
			return
		}
		defer resp.Body.Close()

		buf := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Printf("Error reading data: %v\n", err)
				return
			}
			if n == 0 {
				break
			}
			progressCounters[part] += int64(n)
		}
	}

	// Start the downloads
	progressCounters := make([]int64, *concurrent)
	for i := int64(0); i < *concurrent; i++ {
		wg.Add(1)
		go downloadPart(i, progressCounters)
	}

	// If duration is specified, stop the test after the specified time
	if *duration > 0 {
		go func() {
			time.Sleep(time.Duration(*duration) * time.Second)
			close(done)
		}()
	}

	// Wait for all goroutines to finish or duration to elapse or interrupt signal
	go func() {
		wg.Wait()
		close(done)
	}()

	// Update progress bars
	if *progress {
		go func() {
			for {
				select {
				case <-ticker.C:
					for i := 0; i < int(*concurrent); i++ {
						//						fmt.Println(progressCounters[i])
						displayProgress(i, progressCounters, fileSize/int64(*concurrent))
					}
				case <-done:
					return
				}
			}
		}()
	}

	select {
	case <-done:
	case <-interrupt:
		fmt.Println("\nInterrupt signal received. Stopping the test...")
	}

	elapsed := time.Since(start)
	downloadSpeedBytes := float64(fileSize) / elapsed.Seconds()
	downloadSpeedMBytes := downloadSpeedBytes / (1024 * 1024)

	// Print the summary
	fmt.Printf("Summary:\n")
	fmt.Printf("File URL: %s\n", *target)
	fmt.Printf("File Size: %d bytes\n", fileSize)
	fmt.Printf("Concurrent Downloads: %d\n", *concurrent)
	fmt.Printf("Download Time: %s\n", elapsed)
	fmt.Printf("Download Speed: %.2f bytes/sec (%.2f MB/sec)\n", downloadSpeedBytes, downloadSpeedMBytes)
}

// Function to display progress bar
func displayProgress(part int, progressCounters []int64, total int64) {
	const barWidth = 40
	percent := float64(progressCounters[part]) / float64(total) * 100
	bar := int(percent * barWidth / 100)
	fmt.Printf("\033[%d;0HPart %d: [%-*s] %.2f%%", part+1, part, barWidth, strings.Repeat("=", bar), percent)
}
