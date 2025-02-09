package encoding

type PasswordEncoder interface {
	HashPassword(password string) (string, error)
	CompareHash(passwordHash, password string) bool
}
