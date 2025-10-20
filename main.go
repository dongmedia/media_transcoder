package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"org.donghyuns.com/media/transcoder/pkg"
)

func main() {
	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &opts)
	logger := slog.New(jsonHandler)
	url, fileName, gpuType, preset, isAudio, videoEncoder, audioEncoder, origin, referer, userAgent := InputFileNameAndUrl()

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Start", "Received signal, shutting down...", sig.String())
		cancel()
	}()

	if !isFFmpegInstalled() {
		logger.Info("Start", "FFMPEG is not installed", nil)
		panic("please install ffmpeg first")
	}

	pkg.DownloadWithHeadersAndUserAgent(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent, isAudio)
	<-ctx.Done()
}

func isFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func InputFileNameAndUrl() (string, string, string, string, bool, string, string, string, string, string) {
	var url, fileName, gpuType, preset, audioEncoder, videoEncoder, origin, referer, userAgent string
	var isAudio bool

	log.Println("Input 1.Video URL and 2.Output File Name: ")

	log.Print("1. URL/Video Path: ")
	_, scan1Err := fmt.Scanf("%s", &url)
	if scan1Err != nil {
		log.Fatalf("Scan Error: %v", scan1Err)
	}

	log.Print("2. Output File: ")
	_, scan2Err := fmt.Scanf("%s", &fileName)
	if scan2Err != nil {
		log.Fatalf("Scan File name Error: %v", scan2Err)
	}

	log.Println("3. GPU Usage; nvidia, amd, intel, apple")
	log.Print("You can exclude GPU by inputting empty string")
	_, scan3Err := fmt.Scanf("%s", &gpuType)
	if scan3Err != nil {
		gpuType = ""
		// log.Fatalf("Scan Gpu Type Error: %v", scan3Err)
	}

	log.Println("4. Preset: ultrafast, slow, baseline")
	log.Println("Default: baseline")
	_, scan4Err := fmt.Scanf("%s", &preset)
	if scan4Err != nil {
		preset = "baseline"
		// log.Fatalf("Scan Preset Type Error: %v", scan4Err)
	}

	log.Println("5. Video Encoder:  libx264, libx265, av1, ...")
	log.Println("Default: copy")
	_, scan5Err := fmt.Scanf("%s", &videoEncoder)
	if scan5Err != nil {
		videoEncoder = "copy"
		// log.Fatalf("Scan Video Encoder Type Error: %v", scan5Err)
	}

	log.Println("5. Is Audio Include: true, false")
	log.Println("Default: true")
	_, scan6Err := fmt.Scanf("%b", &isAudio)

	if scan6Err != nil {
		isAudio = true
		// log.Fatalf("Scan is Audio Type Error: %v", scan6Err)
	}

	if isAudio {
		log.Println("6. AudioEncoder: AAC ...")
		log.Println("Default: copy")
		_, scan7Err := fmt.Scanf("%s", &audioEncoder)

		if scan7Err != nil {
			audioEncoder = "copy"
			// log.Fatalf("Scan is Audio Encoder Type Error: %v", scan7Err)
		}
	}

	log.Println("7. Origin Header (for HLS streams, press Enter to skip):")
	log.Println("Example: https://example.com")
	log.Print("Origin: ")
	reader := bufio.NewReader(os.Stdin)
	originInput, _ := reader.ReadString('\n')
	origin = strings.TrimSpace(originInput)

	log.Println("8. Referer Header (for HLS streams, press Enter to skip):")
	log.Println("Example: https://example.com/video.html")
	log.Print("Referer: ")
	refererInput, _ := reader.ReadString('\n')
	referer = strings.TrimSpace(refererInput)

	// Only ask for User-Agent if origin or referer is provided
	if origin != "" || referer != "" {
		log.Println("9. User-Agent Header (press Enter to use default):")
		log.Println("Example: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		log.Print("User-Agent: ")
		userAgentInput, _ := reader.ReadString('\n')
		userAgent = strings.TrimSpace(userAgentInput)
	}

	log.Printf("url: %s\noutputFile: %s\ngpuType: %s, preset: %s\nisAudio: %v, videoEncoder: %s, audioEncoder: %s\norigin: %s, referer: %s, userAgent: %s", url, fileName, gpuType, preset, isAudio, videoEncoder, audioEncoder, origin, referer, userAgent)

	return url, fileName, gpuType, preset, isAudio, videoEncoder, audioEncoder, origin, referer, userAgent
}
