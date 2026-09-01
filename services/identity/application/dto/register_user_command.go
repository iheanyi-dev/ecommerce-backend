package dto

// RegisterUserCommand contains the data required to register a new user.
//
// The command contains primitives because it represents input coming into
// the application boundary. The use case is responsible for converting
// these values into domain types.
type RegisterUserCommand struct {
	FullName string
	Email    string
	Password string
}
