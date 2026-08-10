package user

// RegisterRequest — тело POST /api/user/register.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Salt     string `json:"salt"` // base64, соль для клиентского шифрования
}

// LoginRequest — тело POST /api/user/login.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// AuthResponse — ответ с JWT при успешной аутентификации.
type AuthResponse struct {
	Token string `json:"token"`
	Salt  string `json:"salt"`
}
