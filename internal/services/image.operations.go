package services

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

// CompressParams defines image compression parameters
type CompressParams struct {
	Quality   int
	Format    string
	MaxWidth  int
	MaxHeight int
}

// WatermarkParams defines watermark parameters
type WatermarkParams struct {
	Text     string
	Position string
	Opacity  float64
	FontSize float64
	Color    string // hex color, e.g., "#FFFFFF"
}

// Compress processes resizing and encoding based on CompressParams
func Compress(img image.Image, params CompressParams, out io.Writer) error {
	// Handle Resizing
	if params.MaxWidth > 0 || params.MaxHeight > 0 {
		img = imaging.Fit(img, params.MaxWidth, params.MaxHeight, imaging.Lanczos)
	}

	// Handle Format and Quality
	switch params.Format {
	case "jpeg", "jpg":
		quality := params.Quality
		if quality < 1 || quality > 100 {
			quality = 85
		}
		return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
	case "png":
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		return enc.Encode(out, img)
	default:
		return jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
	}
}

// ApplyWatermark overlays text on the image
func ApplyWatermark(img image.Image, params WatermarkParams) (image.Image, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a new RGBA image to draw on
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	// Create gg context for drawing text
	dc := gg.NewContextForRGBA(dst)

	// Set font size
	fontSize := params.FontSize
	if fontSize < 1 {
		fontSize = 24
	}

	// Load font (using gg's default font)
	// gg.LoadFontFace loads system fonts, fallback to built-in if not found
	dc.LoadFontFace("lato", fontSize)

	// Parse color
	c := parseColor(params.Color)
	alpha := uint8(math.Max(0, math.Min(255, params.Opacity*255)))
	textColor := color.RGBA{c.R, c.G, c.B, alpha}

	dc.SetColor(textColor)

	// Calculate position
	text := params.Text
	tw, th := dc.MeasureString(text)

	var x, y float64
	switch params.Position {
	case "top-left":
		x, y = 20, 20+th
	case "top-right":
		x, y = float64(width)-tw-20, 20+th
	case "bottom-left":
		x, y = 20, float64(height)-20
	case "bottom-right":
		x, y = float64(width)-tw-20, float64(height)-20
	case "center":
		x, y = (float64(width)-tw)/2, (float64(height)+th)/2
	default:
		x, y = float64(width)-tw-20, float64(height)-20
	}

	dc.DrawStringAnchored(text, x, y, 0, 0.5)
	dc.Clip()

	return dst, nil
}

// parseColor converts hex color string to color.RGBA
func parseColor(hex string) color.RGBA {
	if len(hex) == 0 {
		return color.RGBA{255, 255, 255, 255} // default white
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.RGBA{255, 255, 255, 255}
	}

	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}
