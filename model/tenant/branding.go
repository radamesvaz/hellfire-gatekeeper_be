package model

// Logo dimension limits (valid for tenant branding logo).
const (
	LogoMinWidth  = 32
	LogoMinHeight = 32
	LogoMaxWidth  = 512
	LogoMaxHeight = 512
)

// Branding is the DTO returned by GET branding (logo + colors).
type Branding struct {
	LogoURL        string `json:"logo_url"`
	LogoWidth      int    `json:"logo_width"`
	LogoHeight     int    `json:"logo_height"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

// UpdateBrandingColorsRequest is the JSON body for PATCH /auth/tenant/branding/colors.
type UpdateBrandingColorsRequest struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}
