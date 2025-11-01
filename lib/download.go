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

// ---------------------- Config ----------------------

type TranscodeConfig struct {
	URL          string
	OriginalLink string
	OutputFile   string

	GPUType      string // "apple"|"intel"|"amd"|"nvidia"|...
	VideoEncoder string // "", "copy", "libx264","avc1","h264","hevc","libx265","av1","av1_videotoolbox","libaom","libaom-av1","svt","libsvtav1"
	AudioEncoder string // "", "copy", "aac", "libopus" ...
	Preset       string // 소프트웨어 코덱에서만 사용 (x265/svt/libaom)

	IncludeAudio   bool // false면 -an
	EnsureEvenSize bool // true면 홀수 해상도 보정

	// 레이트 컨트롤(옵션)
	UseBitrateTarget bool   // VT에서 q대신 비트레이트 고정
	TargetBitrate    string // "4500k" 등

	// 품질 기본값(옵션, 빈값이면 디폴트)
	VTQualityQ string // 기본 "20"
	X265CRF    string // 기본 "19"
	SVTCRF     string // 기본 "24"
	AOMCRF     string // 기본 "30"

	Prefer10Bit bool // 10-bit 인코딩 지원
}

// ---------------------- Public API ----------------------

// 기존 함수 시그니처 호환 어댑터
func handleTranscodeOptions(url, originalLink, fileName, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool) []string {
	cfg := TranscodeConfig{
		URL:            strings.TrimSpace(url),
		OriginalLink:   originalLink,
		OutputFile:     fileName,
		GPUType:        gpuType,
		VideoEncoder:   videoEncoder,
		AudioEncoder:   audioEncoder,
		Preset:         preset,
		IncludeAudio:   isAudioInclude,
		EnsureEvenSize: false, // 필요시 true로
	}
	return BuildTranscodeArgs(cfg)
}

// 새 모듈화 진입점
func BuildTranscodeArgs(cfg TranscodeConfig) []string {
	var args []string

	// 0) 디코딩 하드웨어 가속(선택)
	args = append(args, hwAccelArgs(cfg.GPUType)...)

	// 1) 입력 / 메타데이터
	args = append(args, "-i", cfg.URL)
	if strings.TrimSpace(cfg.OriginalLink) != "" {
		args = append(args, "-metadata", fmt.Sprintf("url=\"%s\"", cfg.OriginalLink))
	}

	// 2) 비디오 코덱 매핑
	codec := mapCodec(cfg.VideoEncoder)
	if codec != "" {
		args = append(args, "-c:v", codec)
	}

	// 3) 컨테이너 태그 (copy가 아닐 때만)
	if codec != "copy" {
		if tag := tagForCodec(codec); tag != "" {
			args = append(args, "-tag:v", tag)
		}
		// 픽셀 포맷(10bit 지원)
		args = append(args, pixFmtArgs(codec, cfg.Prefer10Bit)...)
		// 공통 호환성 (copy 아닐 때만)
		args = append(args, commonCompatArgs()...)
		// (선택) 홀수 해상도 보정
		if cfg.EnsureEvenSize {
			args = append(args, "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2")
		}
		// 레이트컨트롤
		args = append(args, rateControlArgs(codec, cfg)...)
	}

	// 4) 오디오 처리
	args = append(args, audioArgs(cfg)...)

	// 5) 프리셋 (소프트웨어에서만 의미 있음; VT에서는 무시/미사용 권장)
	//  - 아래 rateControlArgs에서 소프트웨어 코덱별 기본 preset을 이미 채워줌.
	//  - 굳이 사용자 preset을 강제하려면 여기서 보정 가능(주의해서 사용).
	// if strings.TrimSpace(cfg.Preset) != "" && isSoftwareCodec(codec) {
	// 	args = append(args, "-preset", cfg.Preset)
	// }

	// 6) 출력
	args = append(args, "-y", cfg.OutputFile)
	return args
}

// ---------------------- Modules ----------------------

func hwAccelArgs(gpu string) []string {
	switch strings.ToLower(gpu) {
	case "apple":
		return []string{"-hwaccel", "videotoolbox"}
	case "intel":
		return []string{"-hwaccel", "qsv"}
	case "amd":
		// 오타 수정: dxca2 -> dxva2
		return []string{"-hwaccel", "dxva2"}
	case "nvidia":
		return []string{"-hwaccel", "cuda"}
	}
	return nil
}

func mapCodec(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "copy":
		return "copy"
	case "libx264", "avc1", "h264":
		return "h264_videotoolbox"
	case "hevc":
		return "hevc_videotoolbox"
	case "libx265", "x265":
		return "libx265"
	case "av1_videotoolbox", "av1":
		return "av1_videotoolbox"
	case "libaom", "libaom-av1", "aom":
		return "libaom-av1"
	case "svt", "libsvtav1":
		return "libsvtav1"
	default:
		// 알 수 없는 값이면 안전 기본값
		return "h264_videotoolbox"
	}
}

func tagForCodec(codec string) string {
	switch codec {
	case "hevc_videotoolbox", "libx265":
		return "hvc1"
	case "av1_videotoolbox", "libaom-av1", "libsvtav1":
		return "av01"
	default:
		return ""
	}
}

func pixFmtArgs(codec string, prefer10Bit bool) []string {
	// AV1도 10bit 가능(빌드 지원 필요)
	if prefer10Bit {
		switch codec {
		case "hevc_videotoolbox":
			return []string{"-pix_fmt", "p010le"} // VT main10
		case "libx265":
			return []string{"-pix_fmt", "yuv420p10le"} // x265 10bit
		case "libsvtav1", "libaom-av1":
			return []string{"-pix_fmt", "yuv420p10le"} // AV1 10bit
		}
	}
	return []string{"-pix_fmt", "yuv420p"}
}

func commonCompatArgs() []string {
	return []string{"-movflags", "+faststart"}
}

func isSoftwareCodec(codec string) bool {
	return codec == "libx265" || codec == "libsvtav1" || codec == "libaom-av1"
}
func isVideoToolbox(codec string) bool {
	return strings.HasSuffix(codec, "videotoolbox")
}

// mapPreset converts string presets to codec-specific values
func mapPreset(codec, preset string) string {
	p := strings.ToLower(strings.TrimSpace(preset))

	switch codec {
	case "libsvtav1":
		// SVT-AV1 uses numeric presets 0-13 (lower = slower/better quality)
		switch p {
		case "veryslow", "placebo":
			return "3"
		case "slower":
			return "4"
		case "slow":
			return "5"
		case "medium", "":
			return "6"
		case "fast":
			return "7"
		case "faster":
			return "8"
		case "veryfast":
			return "9"
		case "superfast":
			return "10"
		case "ultrafast":
			return "12"
		default:
			// If already numeric, return as-is
			return preset
		}

	case "libaom-av1":
		// libaom-av1 uses cpu-used 0-8 (lower = slower/better quality)
		switch p {
		case "veryslow", "placebo":
			return "1"
		case "slower":
			return "2"
		case "slow":
			return "3"
		case "medium", "":
			return "4"
		case "fast":
			return "5"
		case "faster":
			return "6"
		case "veryfast":
			return "7"
		case "superfast", "ultrafast":
			return "8"
		default:
			// If already numeric, return as-is
			return preset
		}

	case "libx265":
		// x265 accepts string presets, return as-is
		if p == "" {
			return "slow"
		}
		return preset

	default:
		return preset
	}
}

func rateControlArgs(codec string, cfg TranscodeConfig) []string {
	var out []string

	// 기본값
	vtQ := firstNonEmpty(cfg.VTQualityQ, "17")
	x265CRF := firstNonEmpty(cfg.X265CRF, "18")
	svtCRF := firstNonEmpty(cfg.SVTCRF, "24")
	aomCRF := firstNonEmpty(cfg.AOMCRF, "30")

	switch {
	case isSoftwareCodec(codec):
		switch codec {
		case "libx265":
			out = append(out,
				"-crf", x265CRF,
				"-preset", mapPreset(codec, cfg.Preset),
				"-tune", "grain",
				"-x265-params",
				"aq-mode=3:aq-strength=1.0:qcomp=0.72:rd=4:psy-rd=2.0:psy-rdoq=1.0:deblock=-1,-1:strong-intra-smoothing=0:sao=0",
				"-g", "250",
			)

		case "libsvtav1":
			// 고품질/고속 균형 기본값 + 씬컷 감지
			out = append(out,
				"-crf", svtCRF,
				"-preset", mapPreset(codec, cfg.Preset),
				"-g", "300",
				"-svtav1-params", "tune=0:scd=1",
			)

		case "libaom-av1":
			// 속도 옵션: row-mt, 타일, aq-mode
			out = append(out,
				"-crf", aomCRF,
				"-cpu-used", mapPreset(codec, cfg.Preset),
				"-row-mt", "1",
				"-tile-columns", "1", // 2열(1=log2)
				"-tile-rows", "0",
				"-aq-mode", "1",
				"-g", "300",
			)
		}

	case isVideoToolbox(codec):
		// VT: Q 모드 또는 타깃 비트레이트
		if cfg.UseBitrateTarget && strings.TrimSpace(cfg.TargetBitrate) != "" {
			tb := cfg.TargetBitrate
			out = append(out, "-b:v", tb, "-maxrate", tb)
			out = append(out, doubleRateArgs(tb)...)
		} else {
			out = append(out, "-b:v", "0", "-q:v", vtQ)
		}
		out = append(out, "-g", "300")
	}

	return out
}

func audioArgs(cfg TranscodeConfig) []string {
	if !cfg.IncludeAudio {
		return []string{"-an"}
	}
	ae := strings.ToLower(strings.TrimSpace(cfg.AudioEncoder))
	switch ae {
	case "", "copy":
		return []string{"-c:a", "copy"}
	default:
		return []string{"-c:a", cfg.AudioEncoder}
	}
}

func firstNonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func doubleRateArgs(kbps string) []string {
	s := strings.ToLower(strings.TrimSpace(kbps))
	if strings.HasSuffix(s, "k") {
		val := strings.TrimSuffix(s, "k")
		var n int
		fmt.Sscanf(val, "%d", &n)
		if n > 0 {
			return []string{"-bufsize", fmt.Sprintf("%dk", n*2)}
		}
	} else if strings.HasSuffix(s, "m") {
		val := strings.TrimSuffix(s, "m")
		var n int
		fmt.Sscanf(val, "%d", &n)
		if n > 0 {
			return []string{"-bufsize", fmt.Sprintf("%dM", n*2)}
		}
	}
	return []string{}
}
