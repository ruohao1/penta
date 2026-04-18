package cli

import "github.com/google/uuid"

func generateID() string {
	return uuid.NewString()
}
