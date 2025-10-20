package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Download(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool) error {
	return DownloadWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, "", "", isAudio)
}

func DownloadWithHeaders(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudio bool) error {
	return DownloadWithHeadersAndUserAgent(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, "", isAudio)
}

func DownloadWithHeadersAndUserAgent(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent string, isAudio bool) error {
	// urlFormat := filepath.Ext(url)

	if downlaodErr := DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent, isAudio); downlaodErr != nil {
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
	return DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, "", "", "", isAudioInclude)
}

func DownloadHlsViaGpuVideoWithHeaders(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent string, isAudioInclude bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	transCodeOption := handleTranscodeOptionsWithHeaders(url, fileName, gpuType, videoEncoder, audioEncoder, preset, origin, referer, userAgent, isAudioInclude)

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
	return handleTranscodeOptionsWithHeaders(url, fileName, gpuType, videoEncoder, audioEncoder, preset, "", "", "", isAudioInclude)
}

func handleTranscodeOptionsWithHeaders(url, fileName, gpuType, videoEncoder, audioEncoder, preset, origin, referer, userAgent string, isAudioInclude bool) []string {
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

	// Add HTTP headers for HLS streams if provided
	var headers []string
	if origin != "" && strings.TrimSpace(origin) != "" {
		headers = append(headers, "Origin: "+strings.TrimSpace(origin))
	}
	if referer != "" && strings.TrimSpace(referer) != "" {
		headers = append(headers, "Referer: "+strings.TrimSpace(referer))
	}
	if userAgent != "" && strings.TrimSpace(userAgent) != "" {
		headers = append(headers, "User-Agent: "+strings.TrimSpace(userAgent))
	}
	
	if len(headers) > 0 {
		optionList = append(optionList, "-headers", strings.Join(headers, "\r\n"))
	}

	// Input source
	optionList = append(optionList, "-i", strings.Trim(url, " "))

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
		optionList = append(optionList, "-preset", "baseline")
	} else {
		optionList = append(optionList, "-preset", preset)
	}

	// Output options
	optionList = append(optionList, "-y") // Overwrite output file
	optionList = append(optionList, fileName)

	return optionList
}
