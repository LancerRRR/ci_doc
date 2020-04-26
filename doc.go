package doc

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net"
	"reflect"
	"time"
)

var routes []Route
var isRequired bool
var description string
var isRequest bool
var accessed int
var kind string
var packages = make(map[string]bool, 0)
var Mongodb *mgo.Session

type Route struct {
	ID           bson.ObjectId `json:"id" bson:"_id"`
	Description  string        `json:"description" bson:"description"`
	Path         string        `json:"path" bson:"path"`
	Method       string        `json:"method" bson:"method"`
	IsQuery      bool          `json:"isQuery" bson:"isQuery"`
	Service      string        `json:"service" bson:"service"`
	Request      interface{}   `json:"request" bson:"request"`
	Response     interface{}   `json:"response" bson:"response"`
	ResponseJSON interface{}   `json:"responseJSON" bson:"responseJSON"`
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

func UploadRoutes() {
	c := Mongodb.DB("route").C("ci_routes")
	for _, route := range routes {
		if route.Request != nil {
			route.Request = InterfaceToType(route.Request)
		}
		if route.Response != nil {
			route.ResponseJSON = InterfaceToJSON(route.Response)
			route.Response = InterfaceToType(route.Response)
		}
		oldRoute := Route{}
		err := c.Find(bson.M{"service": route.Service, "path": route.Path, "method": route.Method}).One(&oldRoute)
		if err == nil {
			route.ID = oldRoute.ID
			err = c.UpdateId(route.ID, route)
			if err != nil {
				panic(err)
			}
		} else {
			route.ID = bson.NewObjectId()
			err = c.Insert(route)
			if err != nil {
				panic(err)
			}
		}
	}
}

func GetAllRoutes() ([]Route, error) {
	c := Mongodb.DB("route").C("ci_routes")
	out := []Route{}
	err := c.Find(nil).All(&out)
	return out, err
}

//func InitRedis(url string, password string) {
//	cl = redis.NewClient(&redis.Options{
//		Addr:     url,
//		Password: password,
//		DB:       0,
//	})
//}

// allkeys := []string{}
// 	var cursor uint64
// 	for {
// 		var keys []string
// 		var err error
// 		keys, cursor, err = cl.Scan(cursor, "route*", 10).Result()
// 		if err != nil {
// 			return nil, err
// 		}
// 		log.Println(keys)
// 		allkeys = append(allkeys, keys...)
// 		if cursor == 0 {
// 			break
// 		}
// 	}
// 	log.Println(allkeys)
// 	sc, err := cl.MGet(allkeys...).Result()
// 	if err != nil {
// 		return nil, err
// 	}
// 	outputs := []Route{}
// 	for _, v := range sc {
// 		bytes := []byte(v.(string))
// 		output := Route{}
// 		err := bson.Unmarshal(bytes, &output)
// 		if err != nil {
// 			return nil, err
// 		}
// 		outputs = append(outputs, output)
// 	}

func InitMongo(url string) {
	tlsConfig := &tls.Config{}
	tlsConfig.InsecureSkipVerify = true
	dialInfo, err := mgo.ParseURL(url)
	dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
		conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
		return conn, err
	}

	db, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		panic(err)
	}
	Mongodb = db
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
			kind1 := kind
			val := reflect.ValueOf(v)
			typeOfTstObj := val.Type()
			out := make(map[string]interface{}, 0)
			output := RequestNested{}
			output.Description = description
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
			output.Nested = out
			output.Type = kind1
			output.IsRequired = isRequired
			return output
		} else {
			field := Request{}
			field.Type = reflect.TypeOf(v).String()
			field.Description = description
			field.IsRequired = isRequired
			return field
		}
	case reflect.Slice:
		val := reflect.ValueOf(v)
		if val.Type().Elem().Kind() != reflect.Struct || (val.Type().Elem().Kind() == reflect.Ptr && val.Type().Elem().Elem().Kind() == reflect.Struct) {
			field := Request{}
			field.Type = reflect.TypeOf(v).String()
			field.Description = description
			field.IsRequired = isRequired
			return field
		}
		// if val.Len() == 0 {

		// 	field := Request{}
		// 	field.Type = reflect.TypeOf(v).String()
		// 	field.Description = description
		// 	field.IsRequired = isRequired
		// 	return field
		// }
		kind = "array"
		accessed = 1
		return InterfaceToType(reflect.New(val.Type().Elem().Elem()).Interface())
	case reflect.Ptr:
		fmt.Println("ptr: ", reflect.TypeOf(v).String())
		if packages[getPrefix(reflect.TypeOf(v).String())] {
			kind1 := kind
			val := reflect.Indirect(reflect.ValueOf(v))
			fmt.Println("ptr: ", reflect.ValueOf(v))
			typeOfTstObj := val.Type()
			out := make(map[string]interface{}, 0)
			output := RequestNested{}
			output.Description = description
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
				fmt.Println("jsonTag: ", jsonTag)
				key := ""
				for i := 0; i < len(jsonTag); i++ {
					if string(jsonTag[i]) == "," {
						break
					}
					key += string(jsonTag[i])
				}
				out[key] = value
			}
			output.Nested = out
			output.Type = kind1
			output.IsRequired = isRequired
			return output
		} else {
			field := Request{}
			field.Type = reflect.TypeOf(v).String()
			field.Description = description
			field.IsRequired = isRequired
			return field
		}
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
