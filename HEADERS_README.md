# HTTP Headers Support for HLS Streams

This project now supports adding custom HTTP headers (Origin and Referer) when downloading HLS streams. This is useful for accessing streams that require specific headers for authentication or access control.

## New Functions

### In `pkg/download.go`:

1. `DownloadWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudio bool) error`
2. `DownloadWithReconnectionAndHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudio bool, config *ReconnectionConfig) error`
3. `DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudioInclude bool) error`

### In `lib/download.go`:

1. `DownloadWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudio bool) error`
2. `DownloadHlsViaGpuVideoWithHeaders(ctx, url, fileName, gpuType, preset, videoEncoder, audioEncoder, origin, referer string, isAudioInclude bool) error`

## Usage Examples

### Without Headers (Original Functionality)
```go
err := pkg.Download(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", true)
```

### With Origin Header Only
```go
err := pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", "https://example.com", "", true)
```

### With Referer Header Only
```go
err := pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", "", "https://example.com/video.html", true)
```

### With Both Origin and Referer Headers
```go
err := pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", "https://example.com", "https://example.com/video.html", true)
```

### With Empty Headers (Same as No Headers)
```go
err := pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", "", "", true)
```

## How It Works

- **Empty strings**: If origin or referer parameters are empty strings or contain only whitespace, no headers are added
- **Non-empty strings**: Headers are added to the FFmpeg command using the `-headers` option
- **FFmpeg Integration**: Headers are passed to FFmpeg which handles the HTTP requests to the HLS stream
- **Health Checks**: The health check functions also support the custom headers for consistency

## Implementation Details

The headers are added to FFmpeg commands using the `-headers` option:
- Origin header: `-headers "Origin: https://example.com"`
- Referer header: `-headers "Referer: https://example.com/video.html"`
- Both headers: `-headers "Origin: https://example.com\r\nReferer: https://example.com/video.html"`

This ensures that all HLS segment requests include the specified headers, which is essential for streams that require specific referrer policies or CORS headers.