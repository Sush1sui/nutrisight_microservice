package common

import "strings"

func RenameNutrition(arr []map[string]interface{}) []map[string]interface{} {
	for i, item := range arr {
		name := strings.ToLower(item["name"].(string))
		switch name {
		case "fatty acids, total saturated":
			arr[i]["name"] = "Saturated Fats"
		case "fatty acids, total trans":
			arr[i]["name"] = "Trans Fats"
		case "vitamin d (d2 + d3), international units":
			arr[i]["name"] = "Vitamin D2 + D3"
		case "potassium, k":
			arr[i]["name"] = "Potassium"
		case "sodium, na":
			arr[i]["name"] = "Sodium"
		case "calcium, ca":
			arr[i]["name"] = "Calcium"
		case "iron, fe":
			arr[i]["name"] = "Iron"
		case "fiber, total dietary":
			arr[i]["name"] = "Dietary Fiber"
		case "total sugars":
			arr[i]["name"] = "Sugar"
		case "carbohydrate, by difference":
			arr[i]["name"] = "Carbohydrates"
		}
	}
	return arr
}
