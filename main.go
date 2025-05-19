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
	url, fileName, gpuType, preset, isAudio, videoEncoder, audioEncoder := InputFileNameAndUrl()

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

	lib.Download(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, isAudio)
	<-ctx.Done()
}

func isFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func InputFileNameAndUrl() (string, string, string, string, bool, string, string) {
	var url, fileName, gpuType, preset, audioEncoder, videoEncoder string
	var isAudio bool

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
	log.Print("You can exclude GPU by inputting empty string")
	_, scan3Err := fmt.Scanf("%s", &gpuType)
	if scan3Err != nil {
		log.Fatal("Scan Gpu Type Error")
	}

	log.Println("4. Preset: ultrafast, slow, baseline")
	_, scan4Err := fmt.Scanf("%s", &preset)
	if scan4Err != nil {
		log.Fatal("Scan Preset Type Error")
	}

	log.Println("5. Video Encoder:  libx264, libx265, av1, ...")
	log.Println("Default: copy")
	_, scan5Err := fmt.Scanf("%s", &videoEncoder)
	if scan5Err != nil {
		log.Fatal("Scan Video Encoder Type Error")
	}

	log.Println("5. Is Audio Include: true, false")
	_, scan6Err := fmt.Scanf("%b", &isAudio)

	if scan6Err != nil {
		log.Fatal("Scan is Audio Type Error")
	}

	if isAudio {
		log.Println("5. AudioEncoder: AAC ...")
		log.Println("Default: copy")
		_, scan7Err := fmt.Scanf("%s", &audioEncoder)

		if scan7Err != nil {
			log.Fatal("Scan is Audio Encoder Type Error")
		}
	}

	return url, fileName, gpuType, preset, isAudio, videoEncoder, audioEncoder
}
