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
	url, fileName := InputFileNameAndUrl()

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

	log.Printf("FFMPEG is installed")

	lib.Download(ctx, url, fileName)
	<-ctx.Done()
}

func isFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func InputFileNameAndUrl() (string, string) {
	var url, fileName string

	fmt.Sprintln("Input Video URL and Output File Name: ")

	_, scan1Err := fmt.Scanf("%s", &url)
	if scan1Err != nil {
		log.Fatal("Scan Error")
	}

	_, scan2Err := fmt.Scanf("%s", &fileName)
	if scan2Err != nil {
		log.Fatal("Scan File name Error")
	}

	return url, fileName
}
