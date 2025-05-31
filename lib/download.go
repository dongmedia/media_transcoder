package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Download(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool) error {
	// urlFormat := filepath.Ext(url)

	if downlaodErr := DownloadHlsViaGpuVideo(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, isAudio); downlaodErr != nil {
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

func DownloadHlsViaGpuVideo(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudioInclude bool) error {
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
	fileFormat := strings.Split(filepath.Ext(fileName), ".")[1]
	// Hardware Accelation GPU options
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

	// error resilience and skip options
	optionList = append(optionList, "-err_detect", "ignore_err")       // ignore detecting derrors
	optionList = append(optionList, "-fflags", "+igndts")              // ignore DTS error
	optionList = append(optionList, "-avoid_negative_ts", "make_zero") // handle negative timestamp

	optionList = append(optionList, "-i", strings.Trim(url, " "))

	if videoEncoder == "" {
		optionList = append(optionList, "-c:v", "copy")
	} else {
		optionList = append(optionList, "-c:v", videoEncoder)
	}

	if !isAudioInclude {
		optionList = append(optionList, "-an")
	} else {
		// Enhance Audio Stream Error Handling
		if audioEncoder == "copy" {
			// fallback to re-encoding if copy option has error
			optionList = append(optionList, "-c:a", "aac")
			optionList = append(optionList, "-b:a", "128k")
		} else {
			optionList = append(optionList, "-c:a", audioEncoder)
		}
		optionList = append(optionList, "-bsf:a", "aac_adtstoasc") // ADTS -> ASC
	}

	if preset == "" {
		optionList = append(optionList, "-preset", "baseline")
	} else {
		optionList = append(optionList, "-preset", preset)
	}

	optionList = append(optionList, "-benchmark")
	// error output handling
	optionList = append(optionList, "-f", fileFormat)
	optionList = append(optionList, "-movflags", "+faststart")

	optionList = append(optionList, fileName)

	return optionList
}
