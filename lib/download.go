package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Download(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool) error {
	if downlaodErr := transcodeMedia(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, isAudio); downlaodErr != nil {
		log.Printf("Download Url to Video Error: %v", downlaodErr)
		return downlaodErr
	}

	return nil
}

func transcodeMedia(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudioInclude bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	transCodeOption := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

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

func handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool) []string {
	var optionList []string

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

	optionList = append(optionList, "-i", strings.Trim(url, " "))

	if videoEncoder == "" {
		optionList = append(optionList, "-c:v", "copy")
	} else {
		optionList = append(optionList, "-c:v", videoEncoder)
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
