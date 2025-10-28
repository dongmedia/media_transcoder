package lib

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Download(ctx context.Context, url, originalLink, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool) error {
	// urlFormat := filepath.Ext(url)

	if downlaodErr := DownloadHlsViaGpuVideo(ctx, url, originalLink, fileName, gpuType, preset, videoEncoder, audioEncoder, isAudio); downlaodErr != nil {
		log.Printf("Download Url to Video Error: %v", downlaodErr)
		return downlaodErr
	}
	// if urlFormat == ".m3u8" {
	// 	if hlsDownErr := DownloadHlsToVideo(ctx, url, fileName); hlsDownErr != nil {
	// 		log.Printf("Donwload Url to Video Error: %v", hlsDownErr)
	// 		return hlsDownErr
	// 	}
	// } else {
	// 	if downErr := DownloadLink(ctx, url, fileName); downErr != nil {
	// 		log.Printf("Download URL Error: %v", downErr)
	// 		return downErr
	// 	}
	// }

	return nil
}

func DownloadHlsViaGpuVideo(ctx context.Context, url, originalLink, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudioInclude bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	transCodeOption := handleTranscodeOptions(url, originalLink, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	cmd := exec.CommandContext(ctx, ffmpegPath, transCodeOption...)

	start := time.Now()
	// FFmpeg 명령 로깅
	log.Printf("Transcode HLS Stream into Video: %v", fileName)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Transcoding Error (Job %s): %v\n%s", url, err, string(output))

		return err
	}

	log.Printf("Finished: %v", fileName)
	log.Printf("elapsedTime: %v", time.Since(start))

	os.Exit(0) // finish transcoding
	return nil
}

func handleTranscodeOptions(url, originalLink, fileName, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool) []string {
	var optionList []string

	// Hardware acceleration
	switch strings.ToLower(gpuType) {
	case "apple":
		optionList = append(optionList, "-hwaccel", "videotoolbox")
	case "intel":
		optionList = append(optionList, "-hwaccel", "qsv")
	case "amd":
		optionList = append(optionList, "-hwaccel", "dxca2")
	case "nvidia":
		optionList = append(optionList, "-hwaccel", "cuda")
	}

	// Input source
	optionList = append(optionList, "-i", strings.Trim(url, " "))

	if originalLink != "" {
		optionList = append(optionList, "-metadata", fmt.Sprintf("url=\"%s\"", originalLink))
	}

	// Video codec
	if videoEncoder == "" {
		optionList = append(optionList, "-c:v", "copy")
	} else {
		optionList = append(optionList, "-c:v", videoEncoder)
	}

	// Audio handling
	if !isAudioInclude {
		optionList = append(optionList, "-an")
	} else {
		if audioEncoder == "" {
			optionList = append(optionList, "-c:a", "copy")
		} else {
			optionList = append(optionList, "-c:a", audioEncoder)
		}
	}

	// Preset
	if preset == "" {
		optionList = append(optionList, "-preset", "medium")
	} else {
		optionList = append(optionList, "-preset", preset)
	}

	// Output options
	optionList = append(optionList, "-y") // Overwrite output file
	optionList = append(optionList, fileName)

	return optionList
}
