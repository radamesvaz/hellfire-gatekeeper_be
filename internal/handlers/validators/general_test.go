package validators

import "testing"

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{
			name:  "Happy path: valid email",
			email: "validemail@gmail.com",
			valid: true,
		},
		{
			name:  "Happy path: valid email",
			email: "invalidemail@gmail",
			valid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				if valid := IsValidEmail(tt.email); valid == false {
					t.Errorf("expected email to be valid but got invalid")
				}
			} else {
				if valid := IsValidEmail(tt.email); valid == true {
					t.Errorf("expected email to be invalid but got valid")
				}
			}
		})
	}
}
