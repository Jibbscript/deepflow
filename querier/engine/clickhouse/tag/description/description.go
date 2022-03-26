package description

import (
	"errors"
	"fmt"
)

var TAG_DESCRIPTIONS = map[string]map[string][]*TagDescription{
	"flow_log": {
		"l4_flow_log": {},
		"l7_flow_log": {},
	},
}
var TAG_ENUMS = map[string][]*TagEnum{}

type TagDescription struct {
	Name        string
	ClientName  string
	ServerName  string
	DisplayName string
	Type        string
	EnumFile    string
	Category    string
	Operators   string
}

func NewTagDescription(
	name, clientName, serverName, displayName, tagType, enumFile, category, operators string,
) *TagDescription {
	return &TagDescription{
		Name:        name,
		ClientName:  clientName,
		ServerName:  serverName,
		DisplayName: displayName,
		Type:        tagType,
		EnumFile:    enumFile,
		Category:    category,
		Operators:   operators,
	}
}

type TagEnum struct {
	Value       interface{}
	DisplayName interface{}
}

func NewTagEnum(value, displayName interface{}) *TagEnum {
	return &TagEnum{
		Value:       value,
		DisplayName: displayName,
	}
}

func LoadTagDescriptions(tagData map[string]interface{}) {
	// 生成tag description
	for db, tables := range TAG_DESCRIPTIONS {
		tableData, ok := tagData[db]
		if !ok {
			// TODO 记录日志
			continue
		}
		for table := range tables {
			tableTagData, ok := tableData.(map[string]interface{})[table]
			if !ok {
				// TODO 记录日志
				continue
			}
			// 遍历文件内容进行赋值
			for _, tag := range tableTagData.([][]interface{}) {
				description := NewTagDescription(
					tag[0].(string), tag[1].(string), tag[2].(string), tag[3].(string),
					tag[4].(string), tag[5].(string), tag[6].(string), tag[7].(string),
				)
				TAG_DESCRIPTIONS[db][table] = append(TAG_DESCRIPTIONS[db][table], description)
			}
		}
	}

	// 生成tag enum值
	tagEnumData, ok := tagData["enum"]
	if ok {
		for tagName, enumData := range tagEnumData.(map[string]interface{}) {
			tagEnums := []*TagEnum{}
			for _, enumValue := range enumData.([][]interface{}) {
				tagEnums = append(tagEnums, NewTagEnum(enumValue[0], enumValue[1]))
			}
			TAG_ENUMS[tagName] = tagEnums
		}
	}
}

func GetTagDescriptions(db, table string) (map[string][]interface{}, error) {
	dbTagDescriptions, ok := TAG_DESCRIPTIONS[db]
	if !ok {
		return nil, errors.New(fmt.Sprintf("no tag in %s.%s", db, table))
	}
	tableTagDescriptions, ok := dbTagDescriptions[table]
	if !ok {
		return nil, errors.New(fmt.Sprintf("no tag in %s.%s", db, table))
	}

	response := map[string][]interface{}{
		"columns": []interface{}{
			"name", "client_name", "server_name", "display_name", "type", "category", "operators",
		},
		"values": []interface{}{},
	}
	for _, tag := range tableTagDescriptions {
		response["values"] = append(
			response["values"],
			[]interface{}{
				tag.Name, tag.ClientName, tag.ServerName, tag.DisplayName, tag.Type,
				tag.Category, tag.Operators,
			},
		)
	}
	return response, nil
}

func GetTagValues(db, table, tag string) (map[string][]interface{}, error) {
	tagValues, ok := TAG_ENUMS[tag]
	if !ok {
		return nil, errors.New(fmt.Sprintf("tag (%s) not found", tag))
	}

	response := map[string][]interface{}{
		"columns": []interface{}{"value", "display_name"},
		"values":  []interface{}{},
	}
	for _, value := range tagValues {
		response["values"] = append(
			response["values"], []interface{}{value.Value, value.DisplayName},
		)
	}
	return response, nil
}
