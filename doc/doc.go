package doc

import (
	"reflect"
	"time"

	"github.com/go-redis/redis"
	"gopkg.in/mgo.v2/bson"
)

var routes []Route
var isRequired bool
var description string
var isRequest bool
var accessed int
var kind string
var cl *redis.Client
var packages map[string]bool

type Route struct {
	Description  string      `json:"description" bson:"description"`
	Path         string      `json:"path" bson:"path"`
	Method       string      `json:"method" bson:"method"`
	IsQuery      bool        `json:"isQuery" bson:"isQuery"`
	Service      string      `json:"service" bson:"service"`
	Request      interface{} `json:"request" bson:"request"`
	Response     interface{} `json:"response" bson:"response"`
	ResponseJSON interface{} `json:"responseJSON" bson:"responseJSON"`
}

type Request struct {
	Type        string `json:"type" bson:"type"`
	Description string `json:"description" bson:"description"`
	IsRequired  bool   `json:"isRequired" bson:"isRequired"`
}

type Response struct {
	Type        string `json:"type" bson:"type"`
	Description string `json:"description" bson:"description"`
}

type RequestNested struct {
	Type        string      `json:"type" bson:"type"`
	Description string      `json:"description" bson:"description"`
	IsRequired  bool        `json:"isRequired" bson:"isRequired"`
	Nested      interface{} `json:"nested" bson:"nested"`
}

type ResponseNested struct {
	Type        string      `json:"type" bson:"type"`
	Description string      `json:"description" bson:"description"`
	Nested      interface{} `json:"nested" bson:"nested"`
}

func SpecifyPackages(pks []string) {
	for _, v := range pks {
		packages[v] = true
	}
}

func AddRoute(route Route) {
	routes = append(routes, route)
}

func UploadRoutes() error {
	for _, route := range routes {
		if route.Request != nil {
			route.Request = InterfaceToType(route.Request)
		}
		if route.Response != nil {
			route.ResponseJSON = InterfaceToJSON(route.Response)
			route.Response = InterfaceToType(route.Response)
		}
		bytes, _ := bson.Marshal(route)
		err := cl.Set(route.Path, string(bytes), 0).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func GetAllRoutes() ([]Route, error) {
	cl.FlushDB()
	allkeys := []string{}
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = cl.Scan(cursor, "*", 10).Result()
		if err != nil {
			return nil, err
		}
		allkeys = append(allkeys, keys...)
		if cursor == 0 {
			break
		}
	}
	sc, err := cl.MGet(allkeys...).Result()
	if err != nil {
		return nil, err
	}
	outputs := []Route{}
	for _, v := range sc {
		bytes := []byte(v.(string))
		output := Route{}
		err := bson.Unmarshal(bytes, &output)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func InitRedis(url string, password string) {
	cl = redis.NewClient(&redis.Options{
		Addr:     url,
		Password: password,
		DB:       0,
	})
}

func InterfaceToJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case time.Time:
		return "string"
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Struct:
		val := reflect.ValueOf(v)
		typeOfTstObj := val.Type()
		out := make(map[string]interface{}, 0)
		for i := 0; i < val.NumField(); i++ {
			fieldType := val.Field(i)
			jsonTag := typeOfTstObj.Field(i).Tag.Get("json")
			value := InterfaceToJSON(fieldType.Interface())
			key := ""
			for i := 0; i < len(jsonTag); i++ {
				if string(jsonTag[i]) == "," {
					break
				}
				key += string(jsonTag[i])
			}
			out[key] = value
		}
		return out
	case reflect.Slice:
		val := reflect.ValueOf(v)
		if val.Len() == 0 {
			return []interface{}{reflect.TypeOf(v).String()[2:]}
		}
		return []interface{}{InterfaceToJSON(val.Index(0).Interface())}
	default:
		return reflect.TypeOf(v).String()
	}
}

func InterfaceToType(v interface{}) interface{} {
	if v == nil {
		field := Request{}
		field.Type = "interface"
		return field
	}
	// switch v.(type) {
	// case time.Time:
	// 	return "string"
	// }
	switch reflect.TypeOf(v).Kind() {
	case reflect.Struct:
		if packages[getPrefix(reflect.TypeOf(v).String())] {
			accessed1 := accessed
			kind1 := kind
			val := reflect.ValueOf(v)
			typeOfTstObj := val.Type()
			out := make(map[string]interface{}, 0)
			for i := 0; i < val.NumField(); i++ {
				fieldType := val.Field(i)
				isRequired = false
				description = ""
				if typeOfTstObj.Field(i).Tag.Get("required") == "1" {
					isRequired = true
				}
				description = typeOfTstObj.Field(i).Tag.Get("description")
				kind = "object"
				accessed = 1
				value := InterfaceToType(fieldType.Interface())
				jsonTag := typeOfTstObj.Field(i).Tag.Get("json")
				key := ""
				for i := 0; i < len(jsonTag); i++ {
					if string(jsonTag[i]) == "," {
						break
					}
					key += string(jsonTag[i])
				}
				out[key] = value
			}
			if !isRequest {
				output := RequestNested{}
				output.Nested = out
				output.Type = kind1
				output.Description = description
				output.IsRequired = isRequired
				return output
			}
			if accessed1 == 1 {
				output := RequestNested{}
				output.Nested = out
				output.Type = kind1
				output.Description = description
				output.IsRequired = isRequired
				return output
			}
			accessed = 0
			return out
		} else {
			field := Request{}
			field.Type = reflect.TypeOf(v).String()
			field.Description = description
			field.IsRequired = isRequired
			return field
		}
	case reflect.Slice:
		val := reflect.ValueOf(v)
		if val.Len() == 0 {
			field := Request{}
			field.Type = reflect.TypeOf(v).String()
			field.Description = description
			field.IsRequired = isRequired
			return field
		}
		kind = "array"
		accessed = 1
		return InterfaceToType(val.Index(0).Interface())
	default:
		field := Request{}
		field.Type = reflect.TypeOf(v).String()
		field.Description = description
		field.IsRequired = isRequired
		return field
	}
}

func getPrefix(s string) string {
	for i, _ := range s {
		if string(s[i]) == "." {
			return s[:i]
		}
	}
	return ""
}
