-- Drop indexes first (optional but clean)
DROP INDEX IF EXISTS idx_images_status;
DROP INDEX IF EXISTS idx_images_user_id;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS users;
