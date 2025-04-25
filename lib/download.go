package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func Download(ctx context.Context, url, fileName string) error {
	urlFormat := filepath.Ext(url)

	if urlFormat == "m3u8" {
		if hlsDownErr := DownloadHlsToVideo(ctx, url, fileName); hlsDownErr != nil {
			log.Printf("Donwload Url to Video Error: %v", hlsDownErr)
			return hlsDownErr
		}
	} else {
		if downErr := DownloadLink(ctx, url, fileName); downErr != nil {
			log.Printf("Download URL Error: %v", downErr)
			return downErr
		}
	}

	return nil
}

func DownloadLink(ctx context.Context, url, fileName string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", url,
		"-level", "3.0",
		"-c:v", "av1",
		"-c:a", "copy",
		fileName,
	)

	// FFmpeg 명령 로깅
	log.Printf("Transcode Stream into Video: %v", fileName)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()

	if err != nil {

		log.Printf("변환 실패 (Job %s): %v\n%s", url, err, string(output))

		return err
	}
	log.Printf("변환 완료: %v", fileName)

	return nil
}

func DownloadHlsToVideo(ctx context.Context, url, fileName string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 기본값
	}

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", url,
		"-profile:v", "baseline",
		"-level", "3.0",
		"-c:v", "libx264",
		"-c:a", "copy",
		fileName,
	)

	// FFmpeg 명령 로깅
	log.Printf("Transcode HLS Stream into Video: %v", fileName)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("변환 실패 (Job %s): %v\n%s", url, err, string(output))

		return err
	}

	log.Printf("변환 완료: %v", fileName)

	return nil
}
