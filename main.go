package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"org.donghyuns.com/media/transcoder/lib"
)

func main() {
	url, fileName, gpuType := InputFileNameAndUrl()

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %s, shutting down...", sig)
		cancel()
	}()

	if !isFFmpegInstalled() {
		log.Printf("FFMPEG is not installed")
		panic("please install ffmpeg first")
	}

	lib.Download(ctx, url, fileName, gpuType)
	<-ctx.Done()
}

func isFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func InputFileNameAndUrl() (string, string, string) {
	var url, fileName, gpuType string

	log.Println("Input 1.Video URL and 2.Output File Name: ")

	log.Print("1. URL/Video Path: ")
	_, scan1Err := fmt.Scanf("%s", &url)
	if scan1Err != nil {
		log.Fatal("Scan Error")
	}

	log.Print("2. Output File: ")
	_, scan2Err := fmt.Scanf("%s", &fileName)
	if scan2Err != nil {
		log.Fatal("Scan File name Error")
	}

	log.Println("3. GPU Usage; nvidia, amd, intel, apple")
	log.Print("Default GPU Type is apple: ")
	_, scan3Err := fmt.Scanf("%s", &gpuType)
	if scan3Err != nil {
		log.Fatal("Scan Gpu Type Error")
	}
	return url, fileName, gpuType
}
