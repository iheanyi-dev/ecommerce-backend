package errors

import stderrors "errors"

// ErrInvalidEmail indicates that an email address does not satisfy
// the domain's email validation rules.
var ErrInvalidEmail = stderrors.New("invalid email address")

// ErrInvalidFullName indicates that a full name does not satisfy
// the domain's validation rules.
var ErrInvalidFullName = stderrors.New("invalid full name")

// ErrInvalidPasswordHash indicates that a password hash is invalid.
var ErrInvalidPasswordHash = stderrors.New("invalid password hash")

// ErrInvalidRole indicates that an unsupported user role was supplied.
var ErrInvalidRole = stderrors.New("invalid user role")

// ErrInvalidStatus indicates that an unsupported user status was supplied.
var ErrInvalidStatus = stderrors.New("invalid user status")

// ErrInvalidStatusTransition indicates that a requested status change
// violates the domain's allowed status-transition rules.
var ErrInvalidStatusTransition = stderrors.New("invalid user status transition")

// ErrInvalidUserID indicates that a user ID is invalid.
var ErrInvalidUserID = stderrors.New("invalid user ID")

// ErrUserAlreadyVendor indicates that a user who is already a vendor
// cannot be promoted to vendor again.
var ErrUserAlreadyVendor = stderrors.New("user is already a vendor")

// ErrInvalidVendorPromotion indicates that the user does not satisfy
// the domain rules required for vendor promotion.
var ErrInvalidVendorPromotion = stderrors.New("user cannot be promoted to vendor")
