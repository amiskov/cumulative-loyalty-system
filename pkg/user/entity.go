package user

type User struct {
	Id       string `json:"id"`
	Login    string `json:"login"`
	Password []byte `json:"-"`
}
