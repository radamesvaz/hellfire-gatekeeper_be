package auth

type Service interface {
	ComparePasswords(hashPwd string, plainPwd string) error
	CreateJWT(userID uint64, roleID uint64, email string) error
}
