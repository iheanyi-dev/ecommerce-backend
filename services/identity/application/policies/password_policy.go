package policies

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 64
)

var (
	// ErrPasswordTooShort indicates that a password does not contain
	// the minimum required number of characters.
	ErrPasswordTooShort = errors.New("password must contain at least 8 characters")

	// ErrPasswordTooLong indicates that a password exceeds the maximum
	// permitted number of characters.
	ErrPasswordTooLong = errors.New("password must not exceed 64 characters")

	// ErrPasswordMissingUppercase indicates that a password does not
	// contain at least one uppercase letter.
	ErrPasswordMissingUppercase = errors.New("password must contain at least one uppercase letter")

	// ErrPasswordMissingLowercase indicates that a password does not
	// contain at least one lowercase letter.
	ErrPasswordMissingLowercase = errors.New("password must contain at least one lowercase letter")

	// ErrPasswordMissingNumber indicates that a password does not
	// contain at least one numeric character.
	ErrPasswordMissingNumber = errors.New("password must contain at least one number")

	// ErrPasswordMissingSpecialCharacter indicates that a password does
	// not contain at least one character that is neither a letter nor a number.
	ErrPasswordMissingSpecialCharacter = errors.New("password must contain at least one special character")
)

// ValidatePassword validates a plaintext password against the application's
// password-strength policy.
//
// The policy requires:
//   - 8–64 Unicode characters
//   - at least one uppercase letter
//   - at least one lowercase letter
//   - at least one number
//   - at least one special character
//
// Password validation deliberately lives outside the domain. The domain only
// deals with PasswordHash values and therefore never needs to understand
// plaintext password rules.
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)

	if length < MinPasswordLength {
		return ErrPasswordTooShort
	}

	if length > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	var hasUppercase bool
	var hasLowercase bool
	var hasNumber bool
	var hasSpecial bool

	for _, character := range password {
		switch {
		case unicode.IsUpper(character):
			hasUppercase = true
		case unicode.IsLower(character):
			hasLowercase = true
		case unicode.IsNumber(character):
			hasNumber = true
		default:
			// Anything that is neither a letter nor a number is treated
			// as a special character by the agreed password policy.
			hasSpecial = true
		}
	}

	if !hasUppercase {
		return ErrPasswordMissingUppercase
	}

	if !hasLowercase {
		return ErrPasswordMissingLowercase
	}

	if !hasNumber {
		return ErrPasswordMissingNumber
	}

	if !hasSpecial {
		return ErrPasswordMissingSpecialCharacter
	}

	return nil
}
