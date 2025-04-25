package main

import (
	"log"
	"os/exec"
)

func main() {
	if !isFFmpegInstalled() {
		log.Printf("FFMPEG is not installed")
		panic("please install ffmpeg first")
	} else {
		log.Printf("FFMPEG is installed")
	}
}
func isFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
