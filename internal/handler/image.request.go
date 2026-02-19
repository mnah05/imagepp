package handler

import (
	"encoding/json"
	"imagepp/internal/db"
	Job "imagepp/internal/jobs"
	"imagepp/internal/scheduler"
	helpers "imagepp/pkg/helpers"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
)

var validate = validator.New()

type ImageHandler struct {
	log       zerolog.Logger
	queries   *db.Queries
	scheduler *scheduler.Client
}

func NewImageHandler(log zerolog.Logger, queries *db.Queries, scheduler *scheduler.Client) *ImageHandler {
	return &ImageHandler{
		log:       log,
		queries:   queries,
		scheduler: scheduler,
	}
}

type ProcessImageRequest struct {
	Email      string      `json:"email" validate:"required,email"`
	BucketName string      `json:"bucket_name" validate:"required"`
	ImageKey   string      `json:"image_key" validate:"required"`
	Operations []Operation `json:"operations" validate:"required,min=1"`
}

type Operation struct {
	Type   string         `json:"type" validate:"required,oneof=compress watermark"`
	Params map[string]any `json:"params" validate:"required"`
}

type CompressParams struct {
	Quality   int    `json:"quality" validate:"required,min=1,max=100"`
	Format    string `json:"format" validate:"required,oneof=jpeg png webp"`
	MaxWidth  int    `json:"max_width,omitempty" validate:"omitempty,min=1"`
	MaxHeight int    `json:"max_height,omitempty" validate:"omitempty,min=1"`
}

type WatermarkParams struct {
	Text     string  `json:"text" validate:"required"`
	Position string  `json:"position" validate:"required,oneof=top-left top-right bottom-left bottom-right center"`
	Opacity  float64 `json:"opacity" validate:"required,min=0,max=1"`
	FontSize int     `json:"font_size,omitempty" validate:"omitempty,min=1"`
	Color    string  `json:"color,omitempty" validate:"omitempty"`
}

func (h *ImageHandler) ProcessImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ProcessImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error().Err(err).Msg("Failed to decode request")
		helpers.RespondWithError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	defer r.Body.Close()

	if err := validate.Struct(req); err != nil {
		h.log.Error().Err(err).Msg("Validation failed")
		helpers.RespondWithError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	// Check if user exists or create new one
	var user db.User
	var err error
	user, err = h.queries.GetUserByEmail(ctx, req.Email)
	if err == pgx.ErrNoRows {
		user, err = h.queries.CreateUser(ctx, db.CreateUserParams{
			Email:     req.Email,
			CreatedAt: pgtype.Timestamp{Time: time.Now()},
		})

		if err != nil {
			h.log.Error().Err(err).Str("email", req.Email).Msg("Failed to create user")
			helpers.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
		h.log.Info().Int32("user_id", user.ID).Str("email", user.Email).Msg("New User Created")
	} else if err != nil {
		h.log.Error().Err(err).Str("email", req.Email).Msg("Database error getting user")
		helpers.RespondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// TODO: Create image record
	image, err := h.queries.CreateImage(ctx, db.CreateImageParams{
		UserID:     pgtype.Int4{Int32: user.ID, Valid: true},
		BucketName: req.BucketName,
		ImageKey:   req.ImageKey,
		Status: pgtype.Text{
			String: "pending",
			Valid:  true,
		},
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to create image record")
		helpers.RespondWithError(w, http.StatusInternalServerError, "Failed to create image record")
		return
	}
	h.log.Info().
		Int32("image_id", image.ID).
		Int32("user_id", user.ID).
		Str("image_key", req.ImageKey).
		Msg("Image record created")

	// TODO: Enqueue job to scheduler
	operations := make([]map[string]any, len(req.Operations))
	for i, op := range req.Operations {
		operations[i] = map[string]any{
			"type":   op.Type,
			"params": op.Params,
		}
	}

	jobs := Job.Job{
		ImageID:    int64(image.ID),
		UserID:     int64(user.ID),
		BucketName: req.BucketName,
		ImageKey:   req.ImageKey,
		Operations: operations,
	}

	if err := Job.EnqueueImageJob(ctx, h.scheduler, jobs); err != nil {
		h.log.Error().Err(err).Msg("Failed to enqueue image job")
		helpers.RespondWithError(w, http.StatusInternalServerError, "Failed to queue job")
		return
	}

	helpers.RespondWithJSON(w, http.StatusAccepted, helpers.StatusResponse{
		ImageID:    int64(image.ID),
		UserID:     int64(user.ID),
		BucketName: req.BucketName,
		ImageKey:   req.ImageKey,
		Status:     "processing",
	})
}
