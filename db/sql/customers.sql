-- name: ListCustomers :many
SELECT id, name, email, phone, password_hash, created_at, updated_at FROM customers ORDER BY created_at DESC;

-- name: GetCustomerByID :one
SELECT id, name, email, phone, password_hash, created_at, updated_at FROM customers WHERE id = ? LIMIT 1;

-- name: GetCustomerByEmail :one
SELECT id, name, email, phone, password_hash, created_at, updated_at FROM customers WHERE email = ? LIMIT 1;

-- name: CreateCustomer :one
INSERT INTO customers (name, email, phone)
VALUES (?, ?, ?)
RETURNING *;

-- name: CreateCustomerWithPassword :one
INSERT INTO customers (name, email, phone, password_hash)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET name = ?, email = ?, phone = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = ?;
