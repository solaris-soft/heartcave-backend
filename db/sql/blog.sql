-- name: ListPublishedPosts :many
SELECT id, slug, title, created_at FROM posts
WHERE published = 1
ORDER BY created_at DESC;

-- name: GetPostBySlug :one
SELECT * FROM posts
WHERE slug = ? AND published = 1
LIMIT 1;

-- name: ListAllPosts :many
SELECT * FROM posts
ORDER BY created_at DESC;

-- name: GetPostByID :one
SELECT * FROM posts
WHERE id = ?
LIMIT 1;

-- name: CreatePost :one
INSERT INTO posts (slug, title, body, published)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdatePost :one
UPDATE posts
SET slug = ?, title = ?, body = ?, published = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = ?;
