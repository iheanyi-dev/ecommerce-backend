# Identity Service — Architecture & Technical Documentation

## 1. Architecture and Folder Structure

The Identity service is an enterprise-oriented Go microservice built using Clean Architecture.

The dependency direction is:

Presentation → Application → Domain

Infrastructure implements the ports defined by the application layer and integrates external systems.

### Technology Stack

- Go
- `net/http`
- Uber Fx for dependency injection
- PostgreSQL
- pgx / pgxpool
- sqlc for generated database access
- Goose for database migrations
- bcrypt for password hashing
- JWT for access tokens
- OpenTelemetry tracing
- Application/infrastructure logging and metrics

### Folder Structure

```text
services/identity/
├── application/
│   ├── errors/
│   ├── ports/
│   ├── use_cases/
│   └── fx.go
├── domain/
│   ├── entities/
│   ├── errors/
│   ├── value_objects/
│   └── ...
├── infrastructure/
│   ├── observability/
│   ├── persistence/
│   │   └── postgres/
│   │       ├── generated/
│   │       ├── queries/
│   │       └── repositories
│   ├── security/
│   └── fx.go
├── presentation/
│   ├── errors/
│   ├── handlers/
│   ├── middleware/
│   ├── schemas/
│   ├── router.go
│   └── fx.go
├── shared/
│   ├── config/
│   └── fx.go
├── migrations/
├── test/
│   ├── integration/
│   └── unit/
└── cmd/
    └── main.go
Layer Responsibilities
Domain

The domain layer owns business rules and invariants.

It contains:

User aggregate
Value objects
Domain errors
Domain behavior
User state and role rules

The domain has no dependency on:

HTTP
PostgreSQL
JWT
Uber Fx
Infrastructure implementations
Application

The application layer implements use cases and defines the ports required by those use cases.

It coordinates:

Repositories
Password hashing
JWT/token services
Refresh-token services
Logging
Metrics

The application layer depends on abstractions rather than concrete infrastructure implementations.

Infrastructure

The infrastructure layer implements application ports.

It contains:

PostgreSQL repositories
sqlc-generated queries
Password hashing
JWT token service
Refresh-token service
Logging
Metrics
OpenTelemetry tracing
Database lifecycle management
Presentation

The presentation layer exposes the service through HTTP.

It contains:

HTTP handlers
HTTP schemas
Authentication middleware
Authorization middleware
Request observability middleware
Tracing middleware
Router

Presentation is responsible for HTTP concerns and does not own domain business rules.

Shared

The shared layer contains cross-cutting configuration and shared dependency-injection composition.

2. Domain Model and Business Rules
User Aggregate

The central domain object is the User aggregate.

A User contains:

User ID
Full name
Email
Password hash
Role
Account status
Created timestamp
Updated timestamp
Value Objects

Important User value objects include:

UserID
FullName
Email
PasswordHash

Database records are reconstructed through a dedicated domain reconstitution operation rather than bypassing aggregate behavior.

Roles

The Identity service currently recognizes:

admin
vendor
user
Account Statuses

The service currently supports statuses including:

pending_verification
active
suspended
Business Rules
Email is immutable after registration.
Passwords are never stored in plain text.
Only active users can authenticate successfully.
Invalid authentication credentials are handled generically.
Administrators can change a user's account status.
Administrators cannot modify a user's profile through the status-management operation.
Administrative status management cannot promote a user to vendor.
Vendor promotion belongs to the Store workflow.
A user becomes a vendor as part of the future store-creation/business workflow rather than through an administrative status update.
Role and status changes belong to the User domain aggregate.
Authentication and authorization are separate concerns.
Users do not need to be authenticated merely to browse the wider e-commerce platform.
3. Authentication and Authorization Flows
Registration Flow
HTTP Request
     ↓
Register Handler
     ↓
Register User Use Case
     ↓
Validate User Data
     ↓
Check Email Uniqueness
     ↓
Hash Password
     ↓
Create User Aggregate
     ↓
User Repository
     ↓
HTTP Response
Login Flow
HTTP Request
     ↓
Login Handler
     ↓
Authenticate User Use Case
     ↓
Find User by Email
     ↓
Verify Password
     ↓
Check Account Status
     ↓
Generate Access Token
     ↓
Return Authentication Response

Authentication failures use centralized application errors.

Access-Token Authentication

Protected requests pass through authentication middleware.

HTTP Request
     ↓
Authorization Header
     ↓
JWT Validation
     ↓
Extract User Identity + Role
     ↓
Attach Authenticated Identity to Context
     ↓
Authorization Middleware
     ↓
Handler

Missing or invalid access tokens result in authentication failure.

Authorization

Authorization happens after authentication.

Authentication answers:

Who is the caller?

Authorization answers:

Is the caller allowed to perform this operation?

The service distinguishes:

401 Unauthorized — authentication is missing or invalid.
403 Forbidden — authentication succeeded, but the caller lacks the required role.
Protected User Operations

Authenticated users can access:

Current user profile
Current user profile update
Logout
Administrative Operations

Administrators can:

List users
Get a specific user
Update a user's account status

Administrative endpoints require the admin role.

4. JWT and Refresh-Token Lifecycle
Access Tokens

JWT access tokens are generated after successful authentication.

The access token contains the authenticated identity and role information required by the service.

JWT functionality is abstracted behind an application port.

This prevents application use cases from becoming directly coupled to the JWT implementation.

Access Token Validation

The token service validates:

Token structure
Signature
Expiration
Issuer
Required claims
Other configured validation requirements

Invalid, expired, malformed, or otherwise unacceptable tokens are mapped to the centralized invalid-access-token application error.

Refresh Tokens

Refresh tokens allow a user to obtain a new access token without performing another login.

The lifecycle is:

Login
  ↓
Access Token + Refresh Token
  ↓
Refresh Request
  ↓
Validate Refresh Token
  ↓
Rotate Refresh Token
  ↓
Revoke Previous Token
  ↓
Create Replacement Refresh Token
  ↓
Generate New Access Token
  ↓
Return New Authentication Credentials
Refresh-Token Persistence

Refresh tokens are persisted using a hash.

The database stores:

Token ID
User ID
Token hash
Expiration time
Revocation time
Creation time

The raw refresh token is not stored as the persistent credential.

Refresh-Token Security

The lifecycle provides:

Expiration
Revocation
Rotation
Replay protection
Persistent ownership through the User relationship

A revoked refresh token cannot be reused successfully.

A rotated refresh token becomes invalid once it has been replaced.

Persistence failures are translated through centralized application errors.

Logout

Logout requires authentication.

The logout workflow revokes the applicable refresh-token state so that the authentication session cannot continue through the revoked refresh token.

5. Database Schema

The Identity service owns its PostgreSQL database.

Users Table

Migration:

000001_create_users_table.sql

The users table contains:

Column	Type	Rules
id	UUID	Primary key
full_name	VARCHAR(150)	NOT NULL
email	VARCHAR(320)	NOT NULL, UNIQUE
password_hash	TEXT	NOT NULL
role	VARCHAR(20)	Role value
status	VARCHAR(50)	Account status
created_at	TIMESTAMPTZ	Creation timestamp
updated_at	TIMESTAMPTZ	Update timestamp

The unique email constraint prevents duplicate accounts.

Refresh Tokens Table

Migration:

000002_create_refresh_tokens_table.sql

The refresh_tokens table contains:

Column	Type	Rules
id	UUID	Primary key
user_id	UUID	Foreign key to users
token_hash	TEXT	UNIQUE
expires_at	TIMESTAMPTZ	Expiration
revoked_at	TIMESTAMPTZ	Nullable
created_at	TIMESTAMPTZ	Creation timestamp

Relationship:

users
  │
  │ 1
  │
  │
  │ many
  ▼
refresh_tokens

The foreign key uses:

ON DELETE CASCADE

Therefore, refresh-token records are removed when their owning user is deleted.

Indexes

Indexes support:

Email lookup
Refresh-token lookup by user
Refresh-token expiration queries

The existing unique constraint on email already provides a unique database index.

The existing migrations have been reviewed and are considered valid.

Previously applied migrations should not be edited simply to optimize an existing schema.

Future schema changes should be introduced through new migrations.

6. API Endpoints and Error Conventions
Public API
Method	Endpoint	Purpose	Authentication
POST	/api/v1/users/register	Register user	Public
POST	/api/v1/users/login	Authenticate user	Public
POST	/api/v1/users/refresh	Refresh tokens	Public

These endpoints do not require an access token.

Authenticated API
Method	Endpoint	Purpose	Authentication
GET	/api/v1/users/me	Get current user	Required
PATCH	/api/v1/users/me	Update current user profile	Required
POST	/api/v1/users/logout	Logout	Required

All currently supported application roles can access their own authenticated operations.

Administrative API
Method	Endpoint	Purpose	Required Role
GET	/api/v1/admin/users	List users	admin
GET	/api/v1/admin/users/{id}	Get user	admin
PATCH	/api/v1/admin/users/{id}/status	Update account status	admin
Error Architecture

Errors are separated by architectural layer:

Domain Errors
      ↓
Application Errors
      ↓
Presentation Errors
      ↓
HTTP Response

This prevents HTTP-specific concepts from leaking into the domain layer.

Domain Errors

Domain errors represent business-rule violations.

Examples include:

Invalid user data
Invalid status
Invalid status transition
Invalid role-related operation
Application Errors

Application errors represent use-case-level failures.

Examples include:

Invalid credentials
Account not active
Invalid access token
Token-generation failure
Refresh-token persistence failure
Presentation Errors

Presentation maps application/domain failures to appropriate HTTP responses.

The service does not expose internal infrastructure details directly to API clients.

HTTP Status Conventions
200 OK

Successful retrieval or update operations.

201 Created

Successful resource creation.

400 Bad Request

Malformed or invalid client input.

401 Unauthorized

Authentication is missing or invalid.

Examples:

Missing Authorization header
Invalid JWT
Expired JWT
Invalid credentials
Inactive account during authentication
403 Forbidden

Authentication succeeded but the caller is not authorized.

Example:

Authenticated user attempting an admin-only endpoint.
404 Not Found

Requested resource does not exist where applicable.

409 Conflict

Resource conflict such as duplicate registration where applicable.

500 Internal Server Error

Unexpected server-side failure.

Internal implementation details are not exposed.

7. Testing Strategy

The Identity service follows a TDD-oriented testing strategy.

Testing focuses on meaningful behavioral coverage rather than artificially increasing test counts.

Unit Tests

Unit tests cover:

Domain behavior
Value objects
Aggregate rules
Application use cases
Authentication
Authorization
Error classification
Middleware
Security components
Observability behavior

External dependencies are replaced with fakes where appropriate.

Repository Integration Tests

Real PostgreSQL integration tests cover:

User repository
Refresh-token repository
Database connectivity
Persistence behavior
Transactions
Refresh-token rotation
Rollback behavior
Persistence failures
Presentation Tests

HTTP-level tests cover:

Authentication middleware
Authorization middleware
Request handling
HTTP status codes
Error responses
Protected endpoints
Middleware behavior
Fx Validation

The infrastructure Fx module has a dependency-graph validation test.

The service has also been started using the real Fx composition graph.

Successful startup verified construction of:

Configuration
PostgreSQL pool
SQLC queries
User repository
Refresh-token repository
Password hasher
JWT service
Refresh-token service
Logger
Metrics
Tracer
Application use cases
HTTP handlers
Middleware
Router
HTTP server lifecycle
Full Test Suite

The full project suite is run with:

go test ./... -count=1

Tests are normally run without the -v flag.

Verbose output is used only when detailed diagnostics are specifically required.

Testing Principles

Tests should:

Verify actual behavior
Protect important business rules
Validate security boundaries
Detect regression
Verify integration between architectural layers where necessary

Tests should not exist merely to increase coverage numbers.

8. Dependency Injection and Configuration

The Identity service uses Uber Fx for dependency injection.

Fx Composition

The service composes the following modules:

shared.Module
      ↓
infrastructure.Module
      ↓
application.Module
      ↓
presentation.Module
      ↓
HTTP Server
Shared Module

The shared module provides application configuration.

Configuration includes database and service settings and is loaded from the environment.

The database connection string is derived from the configuration object.

Infrastructure Module

The infrastructure module provides concrete implementations.

It constructs:

PostgreSQL connection pool
SQLC queries
User repository
Refresh-token repository
bcrypt password hasher
JWT token service
Refresh-token service
Production logger
Metrics
OpenTelemetry tracer

Infrastructure implementations are exposed through application ports.

Application Module

The application module constructs use cases.

Use cases receive dependencies through interfaces such as:

UserRepository
RefreshTokenRepository
PasswordHasher
TokenService
RefreshTokenService
Logger
Metrics
Tracer

This allows application logic to remain independent of concrete infrastructure.

Presentation Module

The presentation module constructs:

Register handler
Login handler
Refresh handler
Me handler
Profile-update handler
Logout handler
List-users handler
Get-user handler
Update-status handler
Authentication middleware
Authorization middleware
Request observability middleware
Tracing middleware
Router
Main Application

The application composition root is responsible for assembling all modules.

The HTTP server is registered through an Fx lifecycle.

Startup
Fx Application
      ↓
Dependency Construction
      ↓
Lifecycle Hooks
      ↓
HTTP Server Starts
Shutdown
Fx Shutdown
      ↓
HTTP Server Shutdown
      ↓
Infrastructure Lifecycle Cleanup

This keeps application lifecycle management outside business logic.

9. Security Decisions
Password Hashing

Passwords are hashed using bcrypt.

Plain-text passwords are never stored in PostgreSQL.

The password hasher is abstracted behind an application port.

JWT Security

JWT access tokens are centrally generated and validated.

The service validates token integrity and configured claims such as:

Signature
Issuer
Expiration
User identity
Role

The token service remains behind an application abstraction.

Refresh-Token Security

Refresh tokens are stored as hashes.

The service supports:

Expiration
Revocation
Rotation
Replay protection

A revoked refresh token cannot be reused successfully.

Generic Credential Errors

The authentication process intentionally avoids revealing whether a particular email exists.

For example:

Unknown email
Wrong password

are handled through the same general invalid-credentials behavior.

This reduces user/account enumeration risk.

Inactive Accounts

Only active accounts can authenticate.

Pending or suspended accounts are rejected.

Authentication vs Authorization

The service deliberately separates:

Authentication
=
Who are you?

Authorization
=
Are you allowed to do this?

Authentication establishes the user's identity.

Authorization evaluates the authenticated user's role against the requested operation.

Role and Status Separation

Role and account status represent different concepts.

For example:

Role:
admin
vendor
user

Status:
pending_verification
active
suspended

An administrator changing a user's status does not automatically change the user's role.

Vendor Promotion

Vendor promotion is deliberately not implemented as an administrator status operation.

The business rule is:

User
  ↓
Creates Store
  ↓
Store workflow determines vendor relationship

This keeps vendor promotion tied to the Store domain instead of incorrectly coupling it to administrative account-status management.

Least Privilege

Administrative endpoints require the admin role.

Regular users cannot access administrative user-management operations.

Self-service endpoints operate on the authenticated user's own account.

Security Boundary

Security-sensitive behavior is centralized in:

Domain rules
Application use cases
Authentication middleware
Authorization middleware
Password hashing service
JWT token service
Refresh-token service
Refresh-token repository

Handlers are not allowed to become an alternative source of security rules.

Current Identity Service Status

The Identity service has completed the major implementation work through Phase 15.

Completed phases include:

Authentication foundation
Authentication request context
JWT authentication middleware
Authorization middleware
Refresh-token lifecycle
Logout and revocation
User account management
Administrative user management
Vendor-promotion design/decision
Security hardening
Error architecture
Observability
Database and migration hardening
Integration-test suite
Fx dependency composition

Phase 16 — API Completion is the remaining API-surface review/finalization step.

Phase 17 — Documentation is this document.

The next major service after Identity is the Store service.

The Store service will own the store/vendor business workflow and will provide the business context for vendor promotion th