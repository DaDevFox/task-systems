package query

import (
	"errors"
	"regexp"
	"strings"

	"github.com/DaDevFox/task-systems/user-core/backend/internal/constants"

	"github.com/DaDevFox/hof"
	levenshtein "github.com/ka-weihe/fast-levenshtein"

	pb "github.com/DaDevFox/task-systems/user-core/backend/proto/v1"
)

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

func testSpecifier(test string, against *pb.TextQuery) (bool, error) {
	if against == nil {
		return false, errors.New("can't match against nil")
	}

	valid := false

	wantExactMatch := false
	exactMatch := false
	if against.RegexMatchExactly != nil && *against.RegexMatchExactly != "" {
		wantExactMatch = true
		match, err := regexp.MatchString(*against.RegexMatchExactly, test)
		if err != nil {
			exactMatch = match
		}
	}

	valid = valid &&
		(against.ContainsRegexMatches == nil ||
			hof.Every(against.ContainsRegexMatches, func(matchstr string) bool {
				found, err := regexp.MatchString(matchstr, test)
				return err != nil && found
			})) && (!wantExactMatch || exactMatch)
	return valid, nil
}

// TODO: err reporting
func TestUserQuery(req *pb.UserQuery, user *pb.User) bool {
	if req == nil || user == nil {
		return false
	}
	if req.GetQuery() == nil {
		return false
	}

	switch req.GetQuery().(type) {
	case *pb.UserQuery_Join:
		switch *req.GetJoin().Type {
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
			username, err := UserName(user)
			if err != nil {
				return false // WARN: error consumed without emission
			}

			match, err := testSpecifier(username, terminal.Name)
			if err == nil {
				// TODO: log err here at trace level (err tryign to eval one, rolling to next)
				valid = valid && match
			}
		}

		if terminal.Id != nil {
			if user.Id == nil {
				return false
			}
			id := *user.Id

			match, err := testSpecifier(id, terminal.Id)
			if err == nil {
				// TODO: log err here at trace level (err tryign to eval one, rolling to next)
				valid = valid && match
			}
		}
		return valid
	}
	// TODO: ret malformed req err here
	return false
}

func testApproximateSpecifier(test string, against *pb.ApproximateTextQuery, behavior *pb.ApproximationBehavior) (bool, error) {
	if against == nil {
		return false, errors.New("can't match against nil") // TODO: standardized errors like "invalid args" (functional failed_precondition) here, versus failed_precondition (environment/non-arg issues with setup)
	}

	valid := false

	maxDist := behavior.EditDistance.MaxLevenshteinDistance
	wantSimilarTo := maxDist != nil && *maxDist != 0
	matchedSimilarTo := false

	if len(against.SimilarTo) > 0 {
		matchedSimilarTo = hof.Every(against.SimilarTo, func(matchstr string) bool {
			if !wantSimilarTo { // didn't want, but it was present: run exact match on data
				return matchstr == test // TODO: consider substring test here?
			}

			return uint32(levenshtein.Distance(test, matchstr)) <= *maxDist
		})
		wantSimilarTo = true // bad readability here, but saving as an optimization (one less nil, len check)
	}

	if wantSimilarTo {
		valid = valid && matchedSimilarTo
	}

	return valid, nil
}

func TestApproximateUserQuery(req *pb.ApproximateUserQuery, user *pb.User) bool {
	if req == nil || user == nil || req.ApproximationBehavior == nil {
		return false
	}
	if req.GetQuery() == nil {
		return false
	}

	switch req.GetQuery().(type) {
	case *pb.ApproximateUserQuery_Join:
		switch *req.GetJoin().Type {
		case pb.JoinType_CONJUNCTION:
			return TestApproximateUserQuery(req.GetJoin().A, user) && TestApproximateUserQuery(req.GetJoin().B, user)
		case pb.JoinType_DISJUNCTION:
			return TestApproximateUserQuery(req.GetJoin().A, user) || TestApproximateUserQuery(req.GetJoin().B, user)
		case pb.JoinType_NOT:
			return !TestApproximateUserQuery(req.GetJoin().A, user)
		}
	case *pb.ApproximateUserQuery_Terminal:
		terminal := req.GetTerminal()
		valid := false
		username, err := UserName(user)
		if err != nil {
			return false // WARN: erro rconsumed without emission
		}

		switch terminal.GetName().(type) {
		case *pb.ApproximateUserSpecifier_ExactName:
			match, err := testSpecifier(username, terminal.GetExactName())

			if err == nil { // TODO: log here
				valid = valid && match
			}
		case *pb.ApproximateUserSpecifier_InexactName:
			match, err := testApproximateSpecifier(username, terminal.GetInexactName(), req.ApproximationBehavior)

			if err == nil { // TODO: log here
				valid = valid && match
			}
		}

		if user.Id == nil {
			return valid
		}
		id := *user.Id

		switch terminal.GetId().(type) {
		case *pb.ApproximateUserSpecifier_ExactId:
			match, err := testSpecifier(id, terminal.GetExactName())

			if err == nil { // TODO: log here
				valid = valid && match
			}
		case *pb.ApproximateUserSpecifier_InexactId:
			match, err := testApproximateSpecifier(id, terminal.GetInexactName(), req.ApproximationBehavior)

			if err == nil { // TODO: log here
				valid = valid && match
			}
		}

		return valid
	}
	return false
}
