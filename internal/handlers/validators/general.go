package validators

import "regexp"

// Agregar las pruebas unitarias
func IsValidEmail(email string) bool {
	re := regexp.MustCompile(`^[\w\.-]+@[\w\.-]+\.\w+$`)
	return re.MatchString(email)
}
