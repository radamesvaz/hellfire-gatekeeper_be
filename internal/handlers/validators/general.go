package validators

import "regexp"

func IsValidEmail(email string) bool {
	re := regexp.MustCompile(`^[\w\.-]+@[\w\.-]+\.\w+$`)
	return re.MatchString(email)
}
