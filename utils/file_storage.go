package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// MaxImageBytes is the maximum allowed decoded image size (default 5MB)
const MaxImageBytes = 5 * 1024 * 1024

// SaveBase64Image decodes a base64 image string (optionally with data URI prefix)
// and writes it to the local uploads directory. Returns the relative path.
func SaveBase64Image(b64 string) (string, error) {
	if b64 == "" {
		return "", errors.New("empty image string")
	}
	// Strip data URI prefix if present
	if idx := strings.Index(b64, ","); idx != -1 && strings.Contains(strings.ToLower(b64[:idx]), "base64") {
		b64 = b64[idx+1:]
	}

	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Try raw std encoding (no padding)
		data, err = base64.RawStdEncoding.DecodeString(b64)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64: %w", err)
		}
	}

	if len(data) > MaxImageBytes {
		return "", fmt.Errorf("image exceeds max size of %d bytes", MaxImageBytes)
	}

	// Simple MIME detection by magic numbers
	ext := detectImageExt(data)
	if ext == "" {
		// default to .bin to avoid incorrect assumptions
		ext = ".bin"
	}

	// Use SHA-256 hash of content for filename to deduplicate
	sum := sha256.Sum256(data)
	name := fmt.Sprintf("%x%s", sum[:], ext)

	// Ensure uploads directory exists (relative to project root)
	uploadsDir := filepath.Join(".", "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Write original file
	relPath := filepath.Join("uploads", name)
	absPath := filepath.Clean(filepath.Join(".", relPath))
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write image: %w", err)
	}

	// Generate thumbnail (max width 400px, preserve aspect ratio)
	_ = generateThumbnail(data, uploadsDir, name, ext)
	return relPath, nil
}

// detectImageExt attempts to detect common image formats by signature
func detectImageExt(data []byte) string {
	if len(data) < 12 {
		return ""
	}
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 {
		return ".jpg"
	}
	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 && data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return ".png"
	}
	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return ".gif"
	}
	// WEBP: RIFF....WEBP
	if string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ".webp"
	}
	return ""
}

// generateThumbnail creates a smaller version of the image next to the original
func generateThumbnail(data []byte, uploadsDir, name, ext string) error {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	// Determine target size (max width 400)
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return errors.New("invalid image dimensions")
	}
	maxW := 400
	if w <= maxW {
		// No need to resize; still create a copy as thumbnail
		maxW = w
	}
	newW := maxW
	newH := int(float64(h) * (float64(newW) / float64(w)))
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)

	// Thumb filename
	thumbName := strings.TrimSuffix(name, ext) + "_thumb" + ".jpg"
	relThumb := filepath.Join("uploads", thumbName)
	absThumb := filepath.Clean(filepath.Join(".", relThumb))

	// Always encode thumbnail as JPEG to save space
	f, err := os.Create(absThumb)
	if err != nil {
		return err
	}
	defer f.Close()
	// Use medium quality
	if format == "png" || format == "gif" || format == "jpeg" {
		if err := jpeg.Encode(f, dst, &jpeg.Options{Quality: 80}); err != nil {
			return err
		}
	} else {
		if err := jpeg.Encode(f, dst, &jpeg.Options{Quality: 80}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteImageAndThumbnail removes an image file and its derived thumbnail
func DeleteImageAndThumbnail(relPath string) error {
	abs := filepath.Clean(filepath.Join(".", relPath))
	_ = os.Remove(abs)
	// Derive thumb path: originalName + _thumb.jpg
	base := filepath.Base(relPath)
	dir := filepath.Dir(relPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	thumb := filepath.Join(dir, name+"_thumb.jpg")
	_ = os.Remove(filepath.Clean(filepath.Join(".", thumb)))
	return nil
}
