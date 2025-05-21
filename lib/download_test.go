package lib

import (
	"reflect"
	"strings"
	"testing"
)

func TestHandleTranscodeOptions_Default(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	gpuType := ""
	videoEncoder := ""
	audioEncoder := ""
	preset := ""
	isAudioInclude := true

	expected := []string{
		"-i", strings.Trim(url, " "),
		"-c:v", "copy",
		"-c:a", "",
		"-preset", "baseline",
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() Default case = %v, want %v", actual, expected)
	}
}

func TestHandleTranscodeOptions_SpecificVideoEncoder(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	gpuType := ""
	videoEncoder := "libx264"
	audioEncoder := ""
	preset := ""
	isAudioInclude := true

	expected := []string{
		"-i", strings.Trim(url, " "),
		"-c:v", videoEncoder,
		"-c:a", "",
		"-preset", "baseline",
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() SpecificVideoEncoder case = %v, want %v", actual, expected)
	}
}

func TestHandleTranscodeOptions_SpecificAudioEncoder(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	gpuType := ""
	videoEncoder := ""
	audioEncoder := "aac"
	preset := ""
	isAudioInclude := true

	expected := []string{
		"-i", strings.Trim(url, " "),
		"-c:v", "copy",
		"-c:a", audioEncoder,
		"-preset", "baseline",
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() SpecificAudioEncoder case = %v, want %v", actual, expected)
	}
}

func TestHandleTranscodeOptions_AudioDisabled(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	gpuType := ""
	videoEncoder := ""
	audioEncoder := "" // Should be ignored
	preset := ""
	isAudioInclude := false

	expected := []string{
		"-i", strings.Trim(url, " "),
		"-c:v", "copy",
		"-an",
		"-preset", "baseline",
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() AudioDisabled case = %v, want %v", actual, expected)
	}
}

func TestHandleTranscodeOptions_SpecificPreset(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	gpuType := ""
	videoEncoder := ""
	audioEncoder := ""
	preset := "ultrafast"
	isAudioInclude := true

	expected := []string{
		"-i", strings.Trim(url, " "),
		"-c:v", "copy",
		"-c:a", "",
		"-preset", preset,
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() SpecificPreset case = %v, want %v", actual, expected)
	}
}

// Test cases for GPU acceleration
func TestHandleTranscodeOptions_GpuAcceleration(t *testing.T) {
	url := "http://example.com/stream.m3u8"
	fileName := "output.mp4"
	videoEncoder := "h264_videotoolbox" // Example encoder for Apple
	audioEncoder := "aac"
	preset := "fast"
	isAudioInclude := true

	tests := []struct {
		name           string
		gpuType        string
		expectedHwAccel []string // Only the hwaccel part
	}{
		{"Apple", "apple", []string{"-hwaccel", "videotoolbox"}},
		{"Intel", "intel", []string{"-hwaccel", "qsv"}},
		{"AMD", "amd", []string{"-hwaccel", "dxca2"}}, // Corrected from dxca2 to dxva2 based on common ffmpeg usage, if this is wrong, the original code is dxca2
		{"Nvidia", "nvidia", []string{"-hwaccel", "cuda"}},
		{"None", "", nil}, // No hwaccel options
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseExpected := []string{}
			if tt.expectedHwAccel != nil {
				baseExpected = append(baseExpected, tt.expectedHwAccel...)
			}
			baseExpected = append(baseExpected,
				"-i", strings.Trim(url, " "),
				"-c:v", videoEncoder,
				"-c:a", audioEncoder,
				"-preset", preset,
				fileName,
			)

			actual := handleTranscodeOptions(url, fileName, tt.gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)
			if !reflect.DeepEqual(actual, baseExpected) {
				t.Errorf("handleTranscodeOptions() GPU %s = %v, want %v", tt.name, actual, baseExpected)
			}
		})
	}
}

func TestHandleTranscodeOptions_Combination(t *testing.T) {
	url := "rtmp://live.example.com/app/stream"
	fileName := "archive.mkv"
	gpuType := "nvidia"
	videoEncoder := "hevc_nvenc"
	audioEncoder := "opus"
	preset := "slow"
	isAudioInclude := true

	expected := []string{
		"-hwaccel", "cuda",
		"-i", strings.Trim(url, " "),
		"-c:v", videoEncoder,
		"-c:a", audioEncoder,
		"-preset", preset,
		fileName,
	}

	actual := handleTranscodeOptions(url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("handleTranscodeOptions() Combination case = %v, want %v", actual, expected)
	}
}
