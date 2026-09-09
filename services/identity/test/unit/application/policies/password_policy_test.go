package policies_test

import (
	"errors"
	"testing"

	"github.com/iheanyi-dev/ecommerce-backend/services/identity/application/policies"
)

func TestValidatePassword_AcceptsValidPassword(t *testing.T) {
	password := "Strong@1"

	err := policies.ValidatePassword(password)

	if err != nil {
		t.Fatalf("expected valid password, got error: %v", err)
	}
}

func TestValidatePassword_RejectsPasswordShorterThanEightCharacters(t *testing.T) {
	password := "Abc@123"

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestValidatePassword_RejectsPasswordLongerThanSixtyFourCharacters(t *testing.T) {
	password := "A" + "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz1234567890@x"

	if len([]rune(password)) <= 64 {
		t.Fatalf("test password must contain more than 64 characters, got %d", len([]rune(password)))
	}

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordTooLong) {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}

func TestValidatePassword_RejectsMissingUppercaseLetter(t *testing.T) {
	password := "strong@123"

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordMissingUppercase) {
		t.Fatalf("expected ErrPasswordMissingUppercase, got %v", err)
	}
}

func TestValidatePassword_RejectsMissingLowercaseLetter(t *testing.T) {
	password := "STRONG@123"

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordMissingLowercase) {
		t.Fatalf("expected ErrPasswordMissingLowercase, got %v", err)
	}
}

func TestValidatePassword_RejectsMissingNumber(t *testing.T) {
	password := "StrongPassword@"

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordMissingNumber) {
		t.Fatalf("expected ErrPasswordMissingNumber, got %v", err)
	}
}

func TestValidatePassword_RejectsMissingSpecialCharacter(t *testing.T) {
	password := "StrongPassword123"

	err := policies.ValidatePassword(password)

	if !errors.Is(err, policies.ErrPasswordMissingSpecialCharacter) {
		t.Fatalf("expected ErrPasswordMissingSpecialCharacter, got %v", err)
	}
}

func TestValidatePassword_RejectsEmptyPassword(t *testing.T) {
	err := policies.ValidatePassword("")

	if !errors.Is(err, policies.ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestValidatePassword_AllowsDifferentSpecialCharacters(t *testing.T) {
	passwords := []string{
		"Strong@1",
		"Strong#1",
		"Strong$1",
		"Strong%1",
		"Strong!1",
		"Strong_1",
		"Strong-1",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			err := policies.ValidatePassword(password)

			if err != nil {
				t.Fatalf("expected password %q to be valid, got %v", password, err)
			}
		})
	}
}

func TestValidatePassword_AllowsUnicodeLettersAndNumbers(t *testing.T) {
	password := "Äbcé@123"

	err := policies.ValidatePassword(password)

	if err != nil {
		t.Fatalf("expected Unicode password to be valid, got %v", err)
	}
}
