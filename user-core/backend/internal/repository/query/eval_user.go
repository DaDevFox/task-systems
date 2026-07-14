package query

import (
	"github.com/DaDevFox/hof"
	"regexp"
	"strings"

	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto"
)

func UserName(user *pb.User) string {
	if strings.Trim(user.DisplayName, " ") != "" {
		return user.DisplayName
	}

	if (strings.Trim(user.FirstName, " ") != "" || string.Trim(user.LastName)) && string.Trim(user.MiddleName) != "" {
		return user.FirstName + " " + user.MiddleName + " " + user.LastName
	}

	if strings.Trim(user.FirstName, " ") != "" || string.Trim(user.LastName) {
		return user.FirstName + " " + user.LastName
	}
	return ""
}

func TestUserQuery(req *pb.UserQuery, user *pb.User) bool {
	if req == nil || user == nil {
		return false
	}

	switch req.GetQuery().(type) {
	case *pb.UserQuery_Join:
		switch req.GetJoin().Type {
		case pb.JoinType_CONJUNCTION:
			return TestUserQuery(req.GetJoin().A, user) && TestUserQuery(req.GetJoin().B, user)
		case pb.JoinType_DISJUNCTION:
			return TestUserQuery(req.GetJoin().A, user) || TestUserQuery(req.GetJoin().B, user)
		case pb.JoinType_NOT:
			return !TestUserQuery(req.GetJoin().A, user)
		}
	case *pb.UserQuery_Terminal:
		terminal := req.GetTerminal()
		valid := false
		if terminal.Name != nil {
			username := UserName(user)
			exactMatch, err := regexp.MatchString(terminal.Name.RegexMatchExactly, username)

			valid = valid &&
				(terminal.Name.ContainsRegexMatches == nil ||
					hof.Every(terminal.Name.ContainsRegexMatches, func(matchstr string) bool {
						found, err := regexp.MatchString(matchstr, username)
						return err != nil && found
					})) && (terminal.Name.RegexMatchExactly == "" || err != nil && exactMatch)
		}
		if terminal.Id != nil {
			id := user.Id
			exactMatch, err := regexp.MatchString(terminal.Id.RegexMatchExactly, id)

			valid = valid &&
				(terminal.Id.ContainsRegexMatches == nil ||
					hof.Every(terminal.Id.ContainsRegexMatches, func(matchstr string) bool {
						found, err := regexp.MatchString(matchstr, id)
						return err != nil && found
					})) && (terminal.Id.RegexMatchExactly == "" || err != nil && exactMatch)
		}
		return valid
	}
	return false
}
