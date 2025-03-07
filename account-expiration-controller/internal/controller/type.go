package controller

type User struct {
	RealName string `json:"realName"`
	UserName string `json:"userName"`
	Active   bool   `json:"active"`
}

type Content struct {
	TotalResults int    `json:"totalResults"`
	Resources    []User `json:"Resources"`
}
