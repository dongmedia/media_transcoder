package lib

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func Download(ctx context.Context, url, fileName string) {
	var wg sync.WaitGroup
	wg.Add(1)

	urlFormat := filepath.Ext(url)

	go func() {
		defer wg.Done()
		if urlFormat == "m3u8" {
			DownloadHlsToVideo(ctx, url, fileName)
		} else {
			DownloadLink(ctx, url, fileName)
		}
	}()

	wg.Wait()
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
		// "-profile:v", "baseline",
		"-level", "3.0",
		// "-c:v libx264 ",
		// "-preset slow ",
		// "-crf 22",
		"-c:v", "av1",
		"-c:a", "copy",
		fileName,
	)

	// FFmpeg 명령 로깅
	log.Printf("FFmpeg 명령: %v", cmd.Args)

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
	log.Printf("FFmpeg 명령: %v", cmd.Args)

	// 명령 실행 및 오류 처리
	output, err := cmd.CombinedOutput()

	if err != nil {

		log.Printf("변환 실패 (Job %s): %v\n%s", url, err, string(output))

		return err
	}

	log.Printf("변환 완료: %v", fileName)

	return nil
}
