package validity

import (
	"errors"
	"strings"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"
	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
)

func StringDeref(ptr *string) string {
	if ptr == nil {
		return "nil"
	}
	return *ptr
}

func UserName(user *pb.User) (string, error) {
	if user.DisplayName != nil && strings.Trim(*user.DisplayName, constants.STRING_TRIMSET) != "" {
		return *user.DisplayName, nil
	}

	if user.FirstName == nil && user.LastName == nil {
		return "", errors.New("User has no display name or first, last name")
	}

	if user.MiddleName != nil && strings.Trim(*user.MiddleName, constants.STRING_TRIMSET) != "" {
		return *user.FirstName + " " + *user.MiddleName + " " + *user.LastName, nil
	} else {
		return *user.FirstName + " " + *user.LastName, nil
	}
}

// Validates the user has a valid name and id
func ValidateUser(user *pb.User) error {
	if user.DisplayName == nil && (user.FirstName == nil || user.LastName == nil) {
		return errors.New("User has no display name or first, last name")
	}

	if user.Id == nil {
		// TODO: validate id too (e.g. should they always be atl 8 digits, alphanumeric only?)
		return nil
	}

	return nil
}
