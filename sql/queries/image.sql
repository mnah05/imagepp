-- name: GetUserByEmail :one
SELECT id, email, created_at 
FROM users 
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, created_at)
VALUES ($1, $2)
RETURNING id, email, created_at;

-- name: CreateImage :one
INSERT INTO images (user_id, bucket_name, image_key, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, bucket_name, image_key, status, created_at, updated_at;

-- name: GetImageByID :one
SELECT id, user_id, bucket_name, image_key, status, created_at, updated_at
FROM images
WHERE id = $1;

-- name: UpdateImageStatus :exec
UPDATE images 
SET status = $1, updated_at = $2
WHERE id = $3;

-- name: GetImagesByUserID :many
SELECT id, user_id, bucket_name, image_key, status, created_at, updated_at
FROM images
WHERE user_id = $1
ORDER BY created_at DESC;