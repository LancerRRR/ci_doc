package model

type Test struct {
	A *string  `json:"ddd"`
	B int64    `json:"dwadad"`
	C []*Test1 `json:"ccc"`
}

type Test1 struct {
	C *int64 `json:"ddd"`
}
