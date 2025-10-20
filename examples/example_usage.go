package example

import (
	"context"
	"log"
	
	"org.donghyuns.com/media/transcoder/pkg"
)

func ExampleUsage() {
	ctx := context.Background()
	
	// Example 1: Download HLS stream without headers (existing functionality)
	err := pkg.Download(ctx, "https://example.com/stream.m3u8", "output.mp4", "apple", "baseline", "", "", true)
	if err != nil {
		log.Printf("Download without headers failed: %v", err)
	}
	
	// Example 2: Download HLS stream with Origin header only
	err = pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output_with_origin.mp4", "apple", "baseline", "", "", "https://example.com", "", true)
	if err != nil {
		log.Printf("Download with Origin header failed: %v", err)
	}
	
	// Example 3: Download HLS stream with both Origin and Referer headers
	err = pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output_with_headers.mp4", "apple", "baseline", "", "", "https://example.com", "https://example.com/video.html", true)
	if err != nil {
		log.Printf("Download with both headers failed: %v", err)
	}
	
	// Example 3b: Download HLS stream with Origin, Referer, and User-Agent headers
	err = pkg.DownloadWithHeadersAndUserAgent(ctx, "https://example.com/stream.m3u8", "output_with_all_headers.mp4", "apple", "baseline", "", "", "https://example.com", "https://example.com/video.html", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", true)
	if err != nil {
		log.Printf("Download with all headers failed: %v", err)
	}
	
	// Example 4: Download HLS stream with Referer header only
	err = pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output_with_referer.mp4", "apple", "baseline", "", "", "", "https://example.com/video.html", true)
	if err != nil {
		log.Printf("Download with Referer header failed: %v", err)
	}
	
	// Example 5: Download HLS stream with empty headers (same as no headers)
	err = pkg.DownloadWithHeaders(ctx, "https://example.com/stream.m3u8", "output_empty_headers.mp4", "apple", "baseline", "", "", "", "", true)
	if err != nil {
		log.Printf("Download with empty headers failed: %v", err)
	}
	
	log.Println("All download examples completed")
}