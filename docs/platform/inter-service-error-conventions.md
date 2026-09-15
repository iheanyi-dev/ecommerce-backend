# Inter-Service Error Conventions

## Purpose

Services must not expose internal domain, repository, database, or
infrastructure errors directly across service boundaries.

Errors crossing a service boundary must use a stable transport-level
representation.

## HTTP

Gateway-facing HTTP APIs use standard HTTP status codes.

Common mappings:

- 400 Bad Request — malformed or invalid client input
- 401 Unauthorized — missing or invalid authentication
- 403 Forbidden — authenticated but not authorized
- 404 Not Found — requested resource does not exist
- 409 Conflict — business/resource conflict
- 422 Unprocessable Entity — semantically invalid input
- 429 Too Many Requests — rate limit exceeded
- 500 Internal Server Error — unexpected server failure
- 502 Bad Gateway — downstream service failure
- 503 Service Unavailable — service temporarily unavailable
- 504 Gateway Timeout — downstream timeout

## Error Response

Public HTTP APIs should return a stable error envelope:

    {
      "error": {
        "code": "USER_NOT_FOUND",
        "message": "user not found"
      }
    }

The `code` is the stable machine-readable identifier.

The `message` is safe for the client to consume.

Internal database errors, stack traces, secrets, credentials, SQL statements,
and infrastructure details must never be returned to clients.

## gRPC

Internal gRPC APIs use standard gRPC status codes.

Examples:

- InvalidArgument
- Unauthenticated
- PermissionDenied
- NotFound
- AlreadyExists
- FailedPrecondition
- ResourceExhausted
- Unavailable
- DeadlineExceeded
- Internal

Business-specific error codes may be carried using structured gRPC error
details where required.

## Domain Errors

Domain errors remain inside the owning service.

Example:

    domain.ErrInvalidStatusTransition

must be translated by the application/presentation boundary into the
appropriate transport error.

Other services must not import another service's domain error package.

## Downstream Errors

A service receiving an error from another service must translate it into
its own application/transport semantics.

Errors must not be blindly re-exposed as internal implementation details.

## Correlation

Every inter-service error must retain the request/correlation ID so the
failure can be traced across services.

The correlation ID belongs in request metadata/headers and logs, not in
the business error message.

## Logging

Expected business failures may be logged at an appropriate informational
or warning level.

Unexpected infrastructure failures should be logged as errors.

Sensitive values must never be logged.

## Versioning

Error codes are part of the service contract.

Existing error codes must remain stable within an API version.

Removing or changing the meaning of an existing error code is a breaking
contract change and requires a new API version when compatibility cannot
otherwise be maintained.

## Gateway Rule

The Gateway must not invent business errors.

It may translate downstream transport failures into Gateway-appropriate
HTTP responses, while preserving stable error codes where appropriate.
