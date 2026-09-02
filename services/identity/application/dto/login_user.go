package dto

// LoginUserCommand contains the credentials supplied by a user
// attempting to authenticate.
//
// These are primitive values because the command represents data entering
// the application boundary. Domain validation and credential verification
// happen inside the application workflow.
type LoginUserCommand struct {
	Email    string
	Password string
}
