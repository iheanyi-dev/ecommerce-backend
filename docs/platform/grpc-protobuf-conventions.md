# gRPC and Protobuf Conventions

## Package Versioning

All internal gRPC APIs use versioned Protobuf packages:

    <service>.v<version>

Example:

    identity.v1

Breaking API changes require a new version.

## Repository Structure

Protobuf contracts live under:

    proto/<service>/v<version>/

Example:

    proto/identity/v1/identity.proto

Generated Go code is produced into the corresponding versioned Go package.

## Service Naming

Each service exposes a clearly named RPC service:

    <ServiceName>Service

Example:

    IdentityService

## RPC Naming

RPC methods use PascalCase and describe the operation:

    GetUser
    CreateUser
    UpdateUser
    DeleteUser

## Request and Response Messages

Each RPC has dedicated request and response messages:

    GetUserRequest
    GetUserResponse

Do not reuse request messages across unrelated operations.

## Field Naming

Protobuf fields use snake_case.

Example:

    user_id
    full_name
    created_at

Generated Go code will expose the corresponding Go naming automatically.

## IDs

Entity identifiers are represented as strings at the transport boundary.

The receiving service is responsible for validating and converting them
into its domain-specific value objects.

## Errors

Business and domain errors are not encoded as arbitrary strings in normal
responses.

gRPC status codes and structured error details will be used for
inter-service error propagation.

## Compatibility

Existing fields must not be renumbered or reused.

Removed fields must be reserved.

New optional fields should be added using new field numbers.

Breaking changes require a new API version.

## Authentication

Internal gRPC calls must carry service-to-service authentication metadata.

User JWT authentication and service authentication remain separate concerns.

## Correlation

Internal calls must propagate the Gateway/request correlation ID through
gRPC metadata.

The receiving service must preserve that identifier in its logging and
tracing context.

## Implementation Rule

Protobuf contracts define the transport boundary only.

Domain models, repositories, and application use cases must not depend
directly on generated Protobuf types.
