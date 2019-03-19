package filter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

//1.dst必须传指针，而且不能为nil，否则反序列化的时候会失败；
//2.src的json tag最好都加上omitempty，否则后者的空值/零值也会序列化成功，那么将会直接覆盖前者；
//3.所以，在src列表中，第一个src最好是默认配置值，其它src的json tag得带有omitempty
func LoadFormMapOrStruct(dst interface{}, src ...interface{}) error {
	for _, data := range src {
		bs, err := json.Marshal(data)
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(bytes.NewReader(bs))
		decoder.UseNumber()
		err = decoder.Decode(&dst)
		if err != nil {
			return err
		}
	}

	return nil
}

func InArray(elem interface{}, array interface{}) bool {
	arrVal := reflect.ValueOf(array)
	if arrVal.Kind() == reflect.Slice {
		for i := 0; i < arrVal.Len(); i++ {
			if arrVal.Index(i).Interface() == elem {
				return true
			}
		}
	}

	return false
}

func FilterStructResponse(src interface{}, roles []string) error {
	//src必须传入指针（需要是settable）
	if reflect.ValueOf(src).Kind() != reflect.Ptr {
		return fmt.Errorf("response参数类型错误")
	}
	srcType := reflect.TypeOf(src).Elem()
	srcValue := reflect.ValueOf(src).Elem()
	for i := 0; i < srcValue.NumField(); i++ {
		//如果对某字段有权限设置，必须带perm这个tag；
		//  否则认为，对该字段不设置权限；
		perm := srcType.Field(i).Tag.Get("perm")
		if perm == "" {
			continue
		}
		//检查该字段是否为指针类型；
		//  如果是指针类型，需要拿到其底层的真实类型
		var field reflect.Value
		if srcValue.Field(i).Kind() == reflect.Ptr {
			field = srcValue.Field(i).Elem()
		} else {
			field = srcValue.Field(i)
		}
		//如果一个用户拥有多种权限，则以,进行分割
		fieldRoles := strings.Split(perm, ",")
		//查看用户是否有该字段权限
		var hasPerm bool = false
		for _, fieldRole := range fieldRoles {
			if InArray(fieldRole, roles) {
				hasPerm = true
				break
			}
		}
		//如果没有权限，则设置为该字段类型的零值
		//然后，continue
		if hasPerm == false {
			field.Set(reflect.Zero(field.Type()))
			continue
		}
		//如果有权限
		switch field.Kind() {
		case reflect.Struct: //而且该字段是结构体类型，则拿到其地址后，再递归调用本函数
			err := FilterStructResponse(field.Addr().Interface(), roles)
			if err != nil {
				return err
			}
		case reflect.Array,
			reflect.Slice: //而且该字段是数组/切片类型，则具体分析：
			for i := 0; i < field.Len(); i++ {
				//如果field.Index(i)不是struct类型，跳过
				if field.Index(i).Kind() != reflect.Struct {
					break
				}
				//如果field.Index(i)是struct类型，递归调用
				elem := field.Index(i).Addr().Interface()
				err := FilterStructResponse(elem, roles)
				if err != nil {
					return err
				}
			}
		default:
			//其它类型不做处理，正常返回
		}
	}

	return nil
}

type Field struct {
	Perm     []int            `json:"perm"`      //字段权限
	SubField map[string]Field `json:"sub_field"` //字段下的子字段
}

func FilterMapResponse(resp map[string]interface{}, permConf map[string]Field, roles []int) error {
	for name, field := range permConf {
		//查看用户是否有该字段权限
		var hasPerm bool = false
		for _, fieldRole := range field.Perm {
			if InArray(fieldRole, roles) {
				hasPerm = true
				break
			}
		}
		//如果没有权限，则将该字段删除
		//然后，continue
		if hasPerm == false {
			delete(resp, name)
			continue
		}
		//如果有权限
		respValue := reflect.ValueOf(resp[name])
		switch respValue.Kind() {
		case reflect.Map: //而且该字段是map[string]interface{}类型，则递归调用本函数
			data, ok := resp[name].(map[string]interface{})
			if !ok {
				return fmt.Errorf("response参数类型错误")
			}
			err := FilterMapResponse(data, field.SubField, roles)
			if err != nil {
				return fmt.Errorf("response参数类型错误")
			}
		case reflect.Slice,
			reflect.Array: //而且该字段是数组/切片类型，则具体分析：
			for i := 0; i < respValue.Len(); i++ {
				elem := respValue.Index(i)
				//如果elem不是map类型，跳过
				if elem.Kind() != reflect.Map {
					break
				}
				//如果elem是map类型，但不是map[string]interface{}类型报错
				data, ok := elem.Interface().(map[string]interface{})
				if !ok {
					return fmt.Errorf("response参数类型错误")
				}
				//如果elem是map[string]interface{}类型，递归调用
				err := FilterMapResponse(data, field.SubField, roles)
				if err != nil {
					return fmt.Errorf("response参数类型错误")
				}
			}
		}
	}
	return nil
}
