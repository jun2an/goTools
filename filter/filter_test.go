package filter

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLoadFormMapOrStruct(t *testing.T) {
	str := `{"ark_id":"1582712398","uid":232323,"username":"hello_name","video":{"order_id":"126778633","district":"0512","product_id":3},"image":{"order_id":"126778633","judge_result":1}}`

	request := map[string]interface{}{}
	err := json.Unmarshal([]byte(str), &request)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(request)

	resp := map[string]interface{}{}
	err = LoadFormMapOrStruct(&resp, request)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}

func TestInArray(t *testing.T) {
	if InArray("2", []string{"2", "3"}) {
		fmt.Println("yes")
	}
	fmt.Println("no")
}

type tRole struct {
	Id     int    `json:"id" perm:"2"`
	Level  string `json:"level" perm:"3"`
	Status int    `json:"status" perm:"2"`
}

type tUser struct {
	Name   string            `json:"name" perm:"2"`
	Age    int               `json:"age" perm:"2"`
	Phones []string          `json:"phones" perm:"2,3"`
	IdNo   *int64            `json:"id_no" perm:"2"`
	Role   []tRole           `json:"role" perm:"2,3"`
	Ext    map[string]string `json:"ext" perm:"2"`
}

var id = int64(123456)
var role1 = tRole{
	Id:    1,
	Level: "高",
}
var role2 = tRole{
	Id:    2,
	Level: "低",
}
var filter = tUser{
	Name:   "alice",
	Age:    23,
	IdNo:   &id,
	Phones: []string{"15377778888", "17822221111"},
	Role: []tRole{
		role1,
		role2,
	},
	Ext: map[string]string{
		"hello": "world",
	},
}

func TestFilterStructResponse(t *testing.T) {
	err := FilterStructResponse(&filter, []string{"3"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v\n", filter)
	fmt.Println("idNo is: ", *filter.IdNo)
	fmt.Println(filter.Role)
}

const permConf = `
{
	"name":{
		"perm": [1]
	},
	"age":{
		"perm": [1]
	},
	"id_no":{
		"perm": [1]
	},
	"phones":{
		"perm": [1,2]
	},
	"ext":{
		"perm": [1,2]
	},
	"role":{
		"perm": [1,2,3],
		"sub_field": {
			"id": {
				"perm": [1]
			},
			"level": {
				"perm": [2,3]
			}
		}
	}
}
`

func TestFilterMapResponse(t *testing.T) {
	//数据
	resp := map[string]interface{}{}
	err := LoadFormMapOrStruct(&resp, filter)
	if err != nil {
		fmt.Println(err)
	}
	//权限
	perm := map[string]Field{}
	err = json.Unmarshal([]byte(permConf), &perm)
	if err != nil {
		fmt.Println(err)
	}
	//过滤
	err = FilterMapResponse(resp, perm, []int{2, 3})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}
