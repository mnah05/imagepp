package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"imagepp/internal/db"
	"imagepp/internal/jobs"
	"imagepp/internal/services"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
)

// Queries is set by main.go during initialization
var Queries *db.Queries

func HandleImagePP(ctx context.Context, t *asynq.Task) error {
	var p jobs.Job
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal job payload: %w", err)
	}

	// Validate job payload
	if p.ImageID == 0 || p.BucketName == "" || p.ImageKey == "" {
		return fmt.Errorf("invalid job payload: missing required fields")
	}

	// Update status to "processing"
	if err := Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
		ID:     int32(p.ImageID),
		Status: pgtype.Text{String: "processing", Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to update image status to processing: %w", err)
	}

	// Initialize S3 service
	s3Svc, err := services.NewS3Service(ctx, p.BucketName)
	if err != nil {
		_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
			ID:     int32(p.ImageID),
			Status: pgtype.Text{String: "failed", Valid: true},
		})
		return fmt.Errorf("failed to create S3 service: %w", err)
	}
	defer func() {
		// Note: S3Service doesn't have Close method, but if it did, we'd call it here
	}()

	// Download image from S3
	imageData, err := s3Svc.Download(ctx, p.ImageKey)
	if err != nil {
		_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
			ID:     int32(p.ImageID),
			Status: pgtype.Text{String: "failed", Valid: true},
		})
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
			ID:     int32(p.ImageID),
			Status: pgtype.Text{String: "failed", Valid: true},
		})
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Process operations
	for _, op := range p.Operations {
		opType, _ := op["type"].(string)
		params, _ := op["params"].(map[string]any)

		switch opType {
		case "watermark":
			wparams := parseWatermarkParams(params)
			img, err = services.ApplyWatermark(img, wparams)
			if err != nil {
				_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
					ID: int32(p.ImageID),
					Status: pgtype.Text{
						String: "failed",
						Valid:  true,
					},
				})
				return fmt.Errorf("failed to apply watermark: %w", err)
			}
		case "compress":
			// Compression happens during encoding
		}
	}

	// Get compress params (default or from operations)
	compressParams := services.CompressParams{
		Quality: 85,
		Format:  "jpeg",
	}
	for _, op := range p.Operations {
		if opType, _ := op["type"].(string); opType == "compress" {
			compressParams = parseCompressParams(op["params"].(map[string]any))
			break
		}
	}

	// Encode image
	var buf bytes.Buffer
	if err := services.Compress(img, compressParams, &buf); err != nil {
		_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
			ID:     int32(p.ImageID),
			Status: pgtype.Text{String: "failed", Valid: true},
		})
		return fmt.Errorf("failed to encode image: %w", err)
	}

	// Generate output key
	ext := compressParams.Format
	if ext == "jpeg" {
		ext = "jpg"
	}
	outputKey := fmt.Sprintf("processed/%d.%s", p.ImageID, ext)

	// Upload processed image
	if err := s3Svc.Upload(ctx, outputKey, buf.Bytes()); err != nil {
		_ = Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
			ID:     int32(p.ImageID),
			Status: pgtype.Text{String: "failed", Valid: true},
		})
		return fmt.Errorf("failed to upload processed image: %w", err)
	}

	// Note: processed_image_key column doesn't exist in DB, so we log the key
	// In a real implementation, you might want to add this column or create a separate table

	// Update status to "completed"
	if err := Queries.UpdateImageStatus(ctx, db.UpdateImageStatusParams{
		ID:     int32(p.ImageID),
		Status: pgtype.Text{String: "completed", Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to update image status to completed: %w", err)
	}

	return nil
}

func parseWatermarkParams(params map[string]any) services.WatermarkParams {
	wp := services.WatermarkParams{
		Text:     "Watermark",
		Position: "bottom-right",
		Opacity:  0.5,
		FontSize: 24,
		Color:    "#FFFFFF",
	}
	if text, ok := params["text"].(string); ok {
		wp.Text = text
	}
	if pos, ok := params["position"].(string); ok {
		wp.Position = pos
	}
	if opacity, ok := params["opacity"].(float64); ok {
		wp.Opacity = opacity
	}
	if fontSize, ok := params["font_size"].(float64); ok {
		wp.FontSize = fontSize
	} else if fontSizeInt, ok := params["font_size"].(int); ok {
		wp.FontSize = float64(fontSizeInt)
	}
	if color, ok := params["color"].(string); ok {
		wp.Color = color
	}
	return wp
}

func parseCompressParams(params map[string]any) services.CompressParams {
	cp := services.CompressParams{
		Quality:   85,
		Format:    "jpeg",
		MaxWidth:  0,
		MaxHeight: 0,
	}
	if quality, ok := params["quality"].(float64); ok {
		cp.Quality = int(quality)
	} else if qualityInt, ok := params["quality"].(int); ok {
		cp.Quality = qualityInt
	}
	if format, ok := params["format"].(string); ok {
		cp.Format = format
	}
	if maxWidth, ok := params["max_width"].(float64); ok {
		cp.MaxWidth = int(maxWidth)
	} else if maxWidthInt, ok := params["max_width"].(int); ok {
		cp.MaxWidth = maxWidthInt
	}
	if maxHeight, ok := params["max_height"].(float64); ok {
		cp.MaxHeight = int(maxHeight)
	} else if maxHeightInt, ok := params["max_height"].(int); ok {
		cp.MaxHeight = maxHeightInt
	}
	return cp
}
