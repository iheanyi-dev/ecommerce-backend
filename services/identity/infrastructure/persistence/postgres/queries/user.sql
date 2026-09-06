-- name: CreateUser :exec
--
-- Creates a new user in the Identity service database.
--
-- The application layer has already validated the User aggregate before
-- this query is executed. SQLC maps the supplied parameters into the
-- generated CreateUserParams structure.
INSERT INTO users (
    id,
    full_name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
);


-- name: ExistsUserByEmail :one
--
-- Determines whether a user with the supplied email exists.
--
-- We deliberately return a boolean instead of retrieving the complete
-- User record because registration only needs to know whether the email
-- is already taken.
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE email = $1
);

-- name: FindUserByEmail :one
SELECT
    id,
    full_name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: FindUserByID :one
--
-- Retrieves a complete user aggregate using the user's unique identifier.
--
-- The repository translates the returned persistence values into the
-- corresponding domain value objects.
SELECT
    id,
    full_name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- UpdateUserFullName updates only the mutable full-name field.
--
-- Email, password, role, and account status are intentionally excluded
-- because they are not part of the self-service profile update operation.
-- The database timestamp is updated whenever the profile name changes.
-- name: UpdateUserFullName :exec
UPDATE users
SET
    full_name = $2,
    updated_at = NOW()
WHERE id = $1;