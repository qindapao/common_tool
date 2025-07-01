package structutil

import (
	"maps"
	"reflect"
)

// StructToMap 递归转换结构体到 map
func StructToMap(obj any) map[string]any {
	result := make(map[string]any)
	val := reflect.ValueOf(obj)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	// 下面的写法只是为了日志打印，所以更稳妥，需要忽略编译器告警
	//nolint:gocritic
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 处理嵌套结构体
		if fieldValue.Kind() == reflect.Struct {
			if field.Anonymous {
				// **扁平化匿名嵌套结构体（即 Go 结构体继承，如 ParserBase）**
				subMap := StructToMap(fieldValue.Interface())
				// 合并，而不是嵌套
				maps.Copy(result, subMap)
			} else {
				// **真正的嵌套结构体，保持层级**
				result[field.Name] = StructToMap(fieldValue.Interface())
			}
		} else {
			result[field.Name] = fieldValue.Interface()
		}
	}

	return result
}