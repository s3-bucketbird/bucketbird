-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1;

-- name: GetProfileByUserID :one
SELECT * FROM profiles WHERE user_id = $1;

-- name: UpsertProfile :exec
INSERT INTO profiles (id, user_id, first_name, last_name, email, language, timezone, avatar_url, youtube_cookie)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    email = EXCLUDED.email,
    language = EXCLUDED.language,
    timezone = EXCLUDED.timezone,
    avatar_url = EXCLUDED.avatar_url,
    youtube_cookie = EXCLUDED.youtube_cookie,
    updated_at = NOW();

-- name: UpdateYouTubeCookie :exec
UPDATE profiles
SET youtube_cookie = $2, updated_at = NOW()
WHERE user_id = $1;
