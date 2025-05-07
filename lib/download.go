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

func DownloadLink(ctx context.Context, url, fileName string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// fileFormat := filepath.Ext(fileName)

	url = strings.Trim(url, " ")

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", url,
		"-c:v", "libx264",
		"-c:a", "copy",
		fileName,
	)

	// FFmpeg 명령 로깅
	log.Printf("Transcode Stream into Video: %v", fileName)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()

	if err != nil {

		log.Printf("Transcoding Error (Job %s): %v\n%s", url, err, string(output))

		return err
	}
	log.Printf("Finished: %v", fileName)

	return nil
}

func DownloadHlsToVideo(ctx context.Context, url, fileName string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", url,
		"-c:v", "libx264",
		"-c:a", "copy",
		fileName,
	)

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

func DownloadHlsViaGpuVideo(ctx context.Context, url, fileName, gpuType string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if gpuType == "" {
		gpuType = "apple"
	}

	url = strings.Trim(url, " ")
	// fileFormat := filepath.Ext(fileName)

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	var cmd *exec.Cmd

	switch strings.ToLower(gpuType) {
	case "nvidia":
		cmd = exec.CommandContext(ctx, ffmpegPath,
			"-hwaccel", "cuda",
			"-i", url,
			"-c:v", "h264_nvenc",
			"-preset", "fast",
			"-c:a", "copy",
			fileName,
		)
	case "amd":
		cmd = exec.CommandContext(ctx, ffmpegPath,
			"-hwaccel", "dxva2",
			"-i", url,
			"-c:v", "h264_amf",
			"-usage", "transcoding",
			"-c:a", "copy",
			fileName,
		)
	case "intel":
		cmd = exec.CommandContext(ctx, ffmpegPath,
			"-hwaccel", "qsv",
			"-init_hw_device", "qsv=hw",
			"-i", url,
			"-c:v", "h264_qsv",
			"-preset", "faster",
			"-c:a", "copy",
			fileName,
		)
	case "apple":
		cmd = exec.CommandContext(ctx, ffmpegPath,
			"-hwaccel", "videotoolbox",
			// "-hwaccel_output_format", "videotoolbox_vld",
			"-i", url,
			"-c:v", "h264_videotoolbox",
			"-realtime", "true", // 실시간 처리
			"-allow_sw", "1", // 소프트웨어 폴백 허용
			"-b:v", "0", // 품질 우선
			"-c:a", "copy",
			"-movflags", "faststart",
			fileName,
		)
	default:
		// 기본값은 소프트웨어 인코딩
		cmd = exec.CommandContext(ctx, ffmpegPath,
			"-i", url,
			"-c:v", "libx264",
			"-c:a", "copy",
			fileName,
		)
	}

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
