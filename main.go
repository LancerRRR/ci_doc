package main

import (
	"ci-doc/doc"
	"encoding/json"
	"fmt"
)

func main() {
	doc.SpecifyPackages([]string{"*model", "model"})
	// route := &doc.Route{
	// 	ID:          bson.NewObjectId(),
	// 	Description: "测试接口",
	// 	Path:        "/test",
	// 	Method:      "POST",
	// 	Service:     "Test",
	// 	Request:     doc.InterfaceToType(a),
	// }
	b, _ := json.Marshal(doc.InterfaceToType(a))
	fmt.Println(string(b))
}
