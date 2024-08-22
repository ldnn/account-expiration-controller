package controller

type User struct {
	RealName string `json:"realName"`
	Active   bool   `json:"active"`
}

type Content struct {
	TotalResults int    `json:"totalResults"`
	Resources    []User `json:"Resources"`
}
