package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Download(ctx context.Context, url, fileName, gpuType string) error {
	// urlFormat := filepath.Ext(url)

	if downlaodErr := DownloadHlsViaGpuVideo(ctx, url, fileName, gpuType); downlaodErr != nil {
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

func DownloadHlsViaGpuVideo(ctx context.Context, url, fileName, gpuType, videoEncoder, audioEncoder, baseline, preset string, isAudioInclude bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	transCodeOption := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, baseline, preset, isAudioInclude)

	cmd := exec.CommandContext(ctx, ffmpegPath, transCodeOption...)

	// FFmpeg 명령 로깅
	log.Printf("Transcode HLS Stream into Video: %v", fileName)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Transcoding Error (Job %s): %v\n%s", url, err, string(output))

		return err
	}

	log.Printf("Finished: %v", fileName)

	return nil
}

func handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, baseline, preset string, isAudioInclude bool) []string {
	var optionList []string

	switch strings.ToLower(gpuType) {
	case "apple":
		optionList = append(optionList, "-hwaccel", "videotoolbox")
	case "intel":
		optionList = append(optionList, "-hwaccel", "qsv")
	case "amd":
		optionList = append(optionList, "-hwaccel", "dxca2")
	case "nvidia":
		optionList = append(optionList, "cuda")
	}

	optionList = append(optionList, "-i", strings.Trim(url, " "))

	if videoEncoder == "" {
		optionList = append(optionList, "c:v", "copy")
	} else {
		optionList = append(optionList, "c:v", videoEncoder)
	}

	if !isAudioInclude {
		optionList = append(optionList, "-an")
	} else {
		optionList = append(optionList, "-c:a", audioEncoder) // audio encdoer in case of audio included
	}

	if preset == "" {
		optionList = append(optionList, "-preset", "baseline")
	} else {
		optionList = append(optionList, "-preset", preset)
	}

	optionList = append(optionList, fileName)

	return optionList
}
