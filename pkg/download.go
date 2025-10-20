package pkg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ReconnectionConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	HealthCheckTimeout time.Duration
}

func DefaultReconnectionConfig() *ReconnectionConfig {
	return &ReconnectionConfig{
		MaxRetries:      5,
		InitialDelay:    2 * time.Second,
		MaxDelay:        60 * time.Second,
		BackoffFactor:   2.0,
		HealthCheckTimeout: 10 * time.Second,
	}
}

func Download(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool) error {
	return DownloadWithReconnection(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, isAudio, DefaultReconnectionConfig())
}

func DownloadWithHeaders(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudio bool) error {
	return DownloadWithReconnectionAndHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, "", isAudio, DefaultReconnectionConfig())
}

func DownloadWithHeadersAndUserAgent(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent string, isAudio bool) error {
	return DownloadWithReconnectionAndHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent, isAudio, DefaultReconnectionConfig())
}

func DownloadWithReconnection(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudio bool, config *ReconnectionConfig) error {
	return DownloadWithReconnectionAndHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, "", "", "", isAudio, config)
}

func DownloadWithReconnectionAndHeaders(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent string, isAudio bool, config *ReconnectionConfig) error {
	if config == nil {
		config = DefaultReconnectionConfig()
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			
			log.Printf("Reconnection attempt %d/%d after %v delay", attempt, config.MaxRetries, delay)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent, isAudio)
		if err == nil {
			if attempt > 0 {
				log.Printf("Successfully reconnected after %d attempts", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("Download attempt %d failed: %v", attempt+1, err)

		if !isRecoverableError(err) {
			log.Printf("Non-recoverable error detected, stopping reconnection attempts: %v", err)
			break
		}
	}

	return fmt.Errorf("download failed after %d attempts, last error: %v", config.MaxRetries+1, lastErr)
}

func DownloadHlsViaGpuVideo(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder string, isAudioInclude bool) error {
	return DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, "", "", "", isAudioInclude)
}

func DownloadHlsViaGpuVideoWithHeaders(ctx context.Context, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer, userAgent string, isAudioInclude bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url = strings.Trim(url, " ")

	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	// For HLS streams, save directly to single file instead of segments
	// FFmpeg can handle HLS input and output to any format directly

	// For non-HLS streams, use the original method
	transCodeOption := handleTranscodeOptionsWithHeaders(url, fileName, gpuType, videoEncoder, audioEncoder, preset, origin, referer, userAgent, isAudioInclude)
	return executeFFmpegWithStreaming(ctx, ffmpegPath, transCodeOption, url, fileName)
}

func downloadHlsWithSegments(ctx context.Context, url, fileName, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool, ffmpegPath string) error {
	return downloadHlsWithSegmentsAndReconnection(ctx, url, fileName, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude, ffmpegPath, DefaultReconnectionConfig())
}

func downloadHlsWithSegmentsAndReconnection(ctx context.Context, url, fileName, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool, ffmpegPath string, config *ReconnectionConfig) error {
	if config == nil {
		config = DefaultReconnectionConfig()
	}

	// Pre-flight check: Verify HLS stream is accessible with reconnection
	if err := checkHlsStreamHealthWithReconnection(ctx, url, config); err != nil {
		return fmt.Errorf("HLS stream health check failed after reconnection attempts: %v", err)
	}

	// Create output directory for segments
	outputDir := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate HLS segment options
	segmentOptions := handleHlsSegmentOptions(url, outputDir, gpuType, videoEncoder, audioEncoder, preset, isAudioInclude)

	log.Printf("Starting HLS transcoding with real-time segments and reconnection: %v", fileName)

	// Execute FFmpeg with streaming output and reconnection capability
	if err := executeFFmpegWithStreamingAndReconnection(ctx, ffmpegPath, segmentOptions, url, fileName, config); err != nil {
		// Even if FFmpeg fails, we might have partial segments saved
		log.Printf("FFmpeg completed with error after reconnection attempts, but segments may be available in: %s", outputDir)
		return err
	}

	// After successful segmentation, optionally concatenate to single file
	if err := concatenateSegments(ctx, outputDir, fileName, ffmpegPath); err != nil {
		log.Printf("Warning: Failed to concatenate segments, but individual segments are saved in: %s", outputDir)
		return err
	}

	log.Printf("Finished HLS transcoding with reconnection: %v", fileName)
	return nil
}

func checkHlsStreamHealth(ctx context.Context, url string) error {
	return checkHlsStreamHealthWithHeaders(ctx, url, "", "", "")
}

func checkHlsStreamHealthWithHeaders(ctx context.Context, url, origin, referer, userAgent string) error {
	log.Printf("Checking HLS stream health: %s", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set appropriate headers for HLS
	if userAgent != "" && strings.TrimSpace(userAgent) != "" {
		req.Header.Set("User-Agent", strings.TrimSpace(userAgent))
	} else {
		req.Header.Set("User-Agent", "FFmpeg/media_transcoder")
	}
	req.Header.Set("Accept", "application/vnd.apple.mpegurl,application/x-mpegURL,*/*")

	// Add custom headers if provided
	if origin != "" && strings.TrimSpace(origin) != "" {
		req.Header.Set("Origin", strings.TrimSpace(origin))
	}
	if referer != "" && strings.TrimSpace(referer) != "" {
		req.Header.Set("Referer", strings.TrimSpace(referer))
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to HLS stream: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HLS stream returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/vnd.apple.mpegurl") &&
		!strings.Contains(contentType, "application/x-mpegURL") &&
		!strings.Contains(contentType, "text/plain") {
		log.Printf("Warning: Unexpected content type for HLS stream: %s", contentType)
	}

	// Read first few bytes to verify it's a valid m3u8 playlist
	buffer := make([]byte, 256)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read HLS playlist: %v", err)
	}

	content := string(buffer[:n])
	if !strings.HasPrefix(content, "#EXTM3U") {
		return fmt.Errorf("invalid HLS playlist format - missing #EXTM3U header")
	}

	log.Printf("HLS stream health check passed - stream is accessible and valid")
	return nil
}

func executeFFmpegWithStreaming(ctx context.Context, ffmpegPath string, options []string, url, fileName string) error {
	cmd := exec.CommandContext(ctx, ffmpegPath, options...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Progress monitoring
	progressMonitor := &ProgressMonitor{
		lastProgressTime: time.Now(),
		isHLS:            strings.Contains(url, ".m3u8") || strings.Contains(strings.ToLower(url), "hls"),
	}

	// Channel to collect errors from goroutines
	errChan := make(chan error, 3)

	// Context for monitoring goroutines
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stream stdout in real-time with progress monitoring
	go func() {
		defer cancel()
		errChan <- progressMonitor.monitorStdout(stdout)
	}()

	// Stream stderr in real-time for error monitoring
	go func() {
		defer cancel()
		errChan <- progressMonitor.monitorStderr(stderr)
	}()

	// Progress timeout monitor
	go func() {
		defer cancel()
		errChan <- progressMonitor.timeoutMonitor(monitorCtx, cmd)
	}()

	// Wait for command completion or timeout
	cmdErr := cmd.Wait()
	cancel() // Stop monitoring goroutines

	// Collect results from monitoring goroutines
	var streamErrors []error
	for i := 0; i < 3; i++ {
		if streamErr := <-errChan; streamErr != nil {
			streamErrors = append(streamErrors, streamErr)
			log.Printf("Monitor error: %v", streamErr)
		}
	}

	// Check for timeout or stall conditions
	if progressMonitor.hasTimedOut {
		return fmt.Errorf("FFmpeg process timed out - no progress for %v", progressMonitor.getTimeoutDuration())
	}

	if progressMonitor.connectionFailed {
		return fmt.Errorf("FFmpeg failed to connect to HLS stream: %s", url)
	}

	if cmdErr != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", cmdErr)
	}

	return nil
}

type ProgressMonitor struct {
	mu               sync.Mutex
	lastProgressTime time.Time
	lastProgress     string
	hasTimedOut      bool
	connectionFailed bool
	isHLS            bool
}

func (pm *ProgressMonitor) monitorStdout(stdout io.Reader) error {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "time=") || strings.Contains(line, "size=") {
			pm.updateProgress(line)
			log.Printf("Progress: %s", line)
		}
	}
	return scanner.Err()
}

func (pm *ProgressMonitor) monitorStderr(stderr io.Reader) error {
	scanner := bufio.NewScanner(stderr)
	connectionAttempted := false

	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("FFmpeg: %s", line)

		// Check for connection attempts
		if strings.Contains(strings.ToLower(line), "opening") ||
			strings.Contains(strings.ToLower(line), "connection") {
			connectionAttempted = true
			log.Printf("FFmpeg attempting connection...")
		}

		// Monitor for critical errors
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "failed") {
			log.Printf("Detected error in FFmpeg output: %s", line)

			// Check for connection-specific errors
			if strings.Contains(strings.ToLower(line), "connection") ||
				strings.Contains(strings.ToLower(line), "network") ||
				strings.Contains(strings.ToLower(line), "timeout") ||
				strings.Contains(strings.ToLower(line), "unreachable") {
				pm.mu.Lock()
				pm.connectionFailed = true
				pm.mu.Unlock()
			}
		}

		// Check for successful stream detection
		if strings.Contains(strings.ToLower(line), "stream") &&
			(strings.Contains(strings.ToLower(line), "video") || strings.Contains(strings.ToLower(line), "audio")) {
			pm.updateProgress("Stream detected: " + line)
		}
	}

	// If this is HLS and we never saw a connection attempt, mark as failed
	if pm.isHLS && !connectionAttempted {
		pm.mu.Lock()
		pm.connectionFailed = true
		pm.mu.Unlock()
		log.Printf("No connection attempt detected for HLS stream")
	}

	return scanner.Err()
}

func (pm *ProgressMonitor) timeoutMonitor(ctx context.Context, cmd *exec.Cmd) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			pm.mu.Lock()
			timeSinceProgress := time.Since(pm.lastProgressTime)
			timeoutDuration := pm.getTimeoutDuration()
			pm.mu.Unlock()

			if timeSinceProgress > timeoutDuration {
				log.Printf("No progress for %v, killing FFmpeg process", timeSinceProgress)
				pm.mu.Lock()
				pm.hasTimedOut = true
				pm.mu.Unlock()

				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				return fmt.Errorf("progress timeout after %v", timeSinceProgress)
			}

			log.Printf("Progress check: last activity %v ago", timeSinceProgress)
		}
	}
}

func (pm *ProgressMonitor) updateProgress(progress string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.lastProgressTime = time.Now()
	pm.lastProgress = progress
}

func (pm *ProgressMonitor) getTimeoutDuration() time.Duration {
	if pm.isHLS {
		return 30 * time.Second // Shorter timeout for HLS streams
	}
	return 60 * time.Second // Longer timeout for regular streams
}

func concatenateSegments(ctx context.Context, segmentDir, finalOutput, ffmpegPath string) error {
	// Create a file list for concatenation
	listFile := filepath.Join(segmentDir, "filelist.txt")

	// Find all segment files
	files, err := filepath.Glob(filepath.Join(segmentDir, "*.ts"))
	if err != nil {
		return fmt.Errorf("failed to find segment files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no segment files found in %s", segmentDir)
	}

	// Create file list for FFmpeg concat
	f, err := os.Create(listFile)
	if err != nil {
		return fmt.Errorf("failed to create file list: %v", err)
	}
	defer f.Close()

	for _, file := range files {
		fmt.Fprintf(f, "file '%s'\n", file)
	}

	// Concatenate segments
	concatOptions := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y",
		finalOutput,
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, concatOptions...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("concatenation failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Successfully concatenated %d segments into %s", len(files), finalOutput)
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

func handleHlsSegmentOptions(url, outputDir, gpuType, videoEncoder, audioEncoder, preset string, isAudioInclude bool) []string {
	return handleHlsSegmentOptionsWithHeaders(url, outputDir, gpuType, videoEncoder, audioEncoder, preset, "", "", "", isAudioInclude)
}

func handleHlsSegmentOptionsWithHeaders(url, outputDir, gpuType, videoEncoder, audioEncoder, preset, origin, referer, userAgent string, isAudioInclude bool) []string {
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

	// Input
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

	// HLS specific options for real-time segmentation
	optionList = append(optionList,
		"-f", "segment",
		"-segment_time", "10", // 10-second segments
		"-segment_format", "mpegts",
		"-reset_timestamps", "1",
		"-segment_list", filepath.Join(outputDir, "playlist.m3u8"),
		"-segment_list_type", "m3u8",
		"-y",
		filepath.Join(outputDir, "segment_%03d.ts"),
	)

	return optionList
}

func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	
	// Network-related errors that can be recovered
	recoverablePatterns := []string{
		"connection",
		"network",
		"timeout",
		"unreachable",
		"refused",
		"reset",
		"broken pipe",
		"i/o timeout",
		"no route to host",
		"temporary failure",
		"server misbehaving",
		"http2: server sent goaway",
		"eof",
	}

	for _, pattern := range recoverablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Non-recoverable errors
	nonRecoverablePatterns := []string{
		"file not found",
		"no such file",
		"permission denied",
		"invalid argument",
		"malformed",
		"unsupported",
		"codec not found",
		"invalid data",
	}

	for _, pattern := range nonRecoverablePatterns {
		if strings.Contains(errStr, pattern) {
			return false
		}
	}

	// Default to recoverable for unknown errors
	return true
}

func checkHlsStreamHealthWithReconnection(ctx context.Context, url string, config *ReconnectionConfig) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			
			log.Printf("Health check reconnection attempt %d/%d after %v delay", attempt, config.MaxRetries, delay)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		healthCtx, cancel := context.WithTimeout(ctx, config.HealthCheckTimeout)
		err := checkHlsStreamHealth(healthCtx, url)
		cancel()

		if err == nil {
			if attempt > 0 {
				log.Printf("HLS stream health check succeeded after %d reconnection attempts", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("Health check attempt %d failed: %v", attempt+1, err)

		if !isRecoverableError(err) {
			log.Printf("Non-recoverable health check error, stopping attempts: %v", err)
			break
		}
	}

	return fmt.Errorf("HLS stream health check failed after %d attempts, last error: %v", config.MaxRetries+1, lastErr)
}

func executeFFmpegWithStreamingAndReconnection(ctx context.Context, ffmpegPath string, options []string, url, fileName string, config *ReconnectionConfig) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			
			log.Printf("FFmpeg reconnection attempt %d/%d after %v delay", attempt, config.MaxRetries, delay)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Verify stream is still accessible before retrying
			healthCtx, cancel := context.WithTimeout(ctx, config.HealthCheckTimeout)
			if healthErr := checkHlsStreamHealth(healthCtx, url); healthErr != nil {
				cancel()
				log.Printf("Stream health check failed before FFmpeg retry: %v", healthErr)
				lastErr = healthErr
				continue
			}
			cancel()
		}

		// Create a new context for this attempt with enhanced monitoring
		attemptCtx, attemptCancel := context.WithCancel(ctx)
		err := executeFFmpegWithEnhancedMonitoring(attemptCtx, ffmpegPath, options, url, fileName, config)
		attemptCancel()

		if err == nil {
			if attempt > 0 {
				log.Printf("FFmpeg succeeded after %d reconnection attempts", attempt)
			}
			return nil
		}

		lastErr = err
		log.Printf("FFmpeg attempt %d failed: %v", attempt+1, err)

		if !isRecoverableError(err) {
			log.Printf("Non-recoverable FFmpeg error, stopping reconnection attempts: %v", err)
			break
		}
	}

	return fmt.Errorf("FFmpeg failed after %d reconnection attempts, last error: %v", config.MaxRetries+1, lastErr)
}

func executeFFmpegWithEnhancedMonitoring(ctx context.Context, ffmpegPath string, options []string, url, fileName string, config *ReconnectionConfig) error {
	cmd := exec.CommandContext(ctx, ffmpegPath, options...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Enhanced progress monitoring with reconnection awareness
	progressMonitor := &EnhancedProgressMonitor{
		lastProgressTime: time.Now(),
		isHLS:            strings.Contains(url, ".m3u8") || strings.Contains(strings.ToLower(url), "hls"),
		config:           config,
		url:              url,
	}

	// Channel to collect errors from goroutines
	errChan := make(chan error, 4)

	// Context for monitoring goroutines
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stream stdout in real-time with progress monitoring
	go func() {
		defer cancel()
		errChan <- progressMonitor.monitorStdout(stdout)
	}()

	// Stream stderr in real-time for error monitoring
	go func() {
		defer cancel()
		errChan <- progressMonitor.monitorStderr(stderr)
	}()

	// Progress timeout monitor with reconnection detection
	go func() {
		defer cancel()
		errChan <- progressMonitor.enhancedTimeoutMonitor(monitorCtx, cmd)
	}()

	// Connection health monitor
	go func() {
		defer cancel()
		errChan <- progressMonitor.connectionHealthMonitor(monitorCtx)
	}()

	// Wait for command completion or timeout
	cmdErr := cmd.Wait()
	cancel() // Stop monitoring goroutines

	// Collect results from monitoring goroutines
	var streamErrors []error
	for i := 0; i < 4; i++ {
		if streamErr := <-errChan; streamErr != nil {
			streamErrors = append(streamErrors, streamErr)
			log.Printf("Monitor error: %v", streamErr)
		}
	}

	// Check for timeout or stall conditions
	if progressMonitor.hasTimedOut {
		return fmt.Errorf("FFmpeg process timed out - no progress for %v", progressMonitor.getTimeoutDuration())
	}

	if progressMonitor.connectionFailed {
		return fmt.Errorf("FFmpeg connection failed to HLS stream: %s", url)
	}

	if progressMonitor.streamDisconnected {
		return fmt.Errorf("FFmpeg detected stream disconnection: %s", url)
	}

	if cmdErr != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", cmdErr)
	}

	return nil
}

type EnhancedProgressMonitor struct {
	mu                  sync.Mutex
	lastProgressTime    time.Time
	lastProgress        string
	hasTimedOut         bool
	connectionFailed    bool
	streamDisconnected  bool
	isHLS               bool
	config              *ReconnectionConfig
	url                 string
	lastHealthCheck     time.Time
}

func (epm *EnhancedProgressMonitor) monitorStdout(stdout io.Reader) error {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "time=") || strings.Contains(line, "size=") {
			epm.updateProgress(line)
			log.Printf("Progress: %s", line)
		}
	}
	return scanner.Err()
}

func (epm *EnhancedProgressMonitor) monitorStderr(stderr io.Reader) error {
	scanner := bufio.NewScanner(stderr)
	connectionAttempted := false

	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("FFmpeg: %s", line)

		// Check for connection attempts
		if strings.Contains(strings.ToLower(line), "opening") ||
			strings.Contains(strings.ToLower(line), "connection") {
			connectionAttempted = true
			log.Printf("FFmpeg attempting connection...")
		}

		// Monitor for critical errors and disconnections
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "failed") {
			log.Printf("Detected error in FFmpeg output: %s", line)

			// Check for connection-specific errors
			if strings.Contains(strings.ToLower(line), "connection") ||
				strings.Contains(strings.ToLower(line), "network") ||
				strings.Contains(strings.ToLower(line), "timeout") ||
				strings.Contains(strings.ToLower(line), "unreachable") {
				epm.mu.Lock()
				epm.connectionFailed = true
				epm.mu.Unlock()
			}
		}

		// Detect stream disconnections
		if strings.Contains(strings.ToLower(line), "eof") ||
			strings.Contains(strings.ToLower(line), "broken pipe") ||
			strings.Contains(strings.ToLower(line), "connection lost") ||
			strings.Contains(strings.ToLower(line), "stream ended") {
			epm.mu.Lock()
			epm.streamDisconnected = true
			epm.mu.Unlock()
			log.Printf("Stream disconnection detected: %s", line)
		}

		// Check for successful stream detection
		if strings.Contains(strings.ToLower(line), "stream") &&
			(strings.Contains(strings.ToLower(line), "video") || strings.Contains(strings.ToLower(line), "audio")) {
			epm.updateProgress("Stream detected: " + line)
		}
	}

	// If this is HLS and we never saw a connection attempt, mark as failed
	if epm.isHLS && !connectionAttempted {
		epm.mu.Lock()
		epm.connectionFailed = true
		epm.mu.Unlock()
		log.Printf("No connection attempt detected for HLS stream")
	}

	return scanner.Err()
}

func (epm *EnhancedProgressMonitor) enhancedTimeoutMonitor(ctx context.Context, cmd *exec.Cmd) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			epm.mu.Lock()
			timeSinceProgress := time.Since(epm.lastProgressTime)
			timeoutDuration := epm.getTimeoutDuration()
			epm.mu.Unlock()

			if timeSinceProgress > timeoutDuration {
				log.Printf("No progress for %v, will trigger reconnection", timeSinceProgress)
				epm.mu.Lock()
				epm.hasTimedOut = true
				epm.mu.Unlock()

				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				return fmt.Errorf("progress timeout after %v", timeSinceProgress)
			}

			log.Printf("Progress check: last activity %v ago", timeSinceProgress)
		}
	}
}

func (epm *EnhancedProgressMonitor) connectionHealthMonitor(ctx context.Context) error {
	if !epm.isHLS {
		return nil // Only monitor HLS streams
	}

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			epm.mu.Lock()
			lastCheck := epm.lastHealthCheck
			epm.lastHealthCheck = time.Now()
			epm.mu.Unlock()

			// Skip if we've checked recently
			if time.Since(lastCheck) < 25*time.Second {
				continue
			}

			// Perform quick health check
			healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := checkHlsStreamHealth(healthCtx, epm.url)
			cancel()

			if err != nil {
				log.Printf("Background health check failed: %v", err)
				epm.mu.Lock()
				epm.connectionFailed = true
				epm.mu.Unlock()
				return fmt.Errorf("background health check failed: %v", err)
			}

			log.Printf("Background health check passed")
		}
	}
}

func (epm *EnhancedProgressMonitor) updateProgress(progress string) {
	epm.mu.Lock()
	defer epm.mu.Unlock()
	epm.lastProgressTime = time.Now()
	epm.lastProgress = progress
}

func (epm *EnhancedProgressMonitor) getTimeoutDuration() time.Duration {
	if epm.isHLS {
		return 45 * time.Second // Longer timeout for HLS with reconnection
	}
	return 90 * time.Second // Longer timeout for regular streams with reconnection
}
