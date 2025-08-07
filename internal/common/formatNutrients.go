package common

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func FormatNutriments(nutriments map[string]interface{}) []map[string]interface{} {
    mainNutrients := []string{
        "energy-kcal", "fat", "saturated-fat", "trans-fat", "cholesterol",
        "carbohydrates", "sugars", "fiber", "proteins", "salt", "sodium",
        "vitamin-a", "vitamin-c", "vitamin-d", "calcium", "iron", "potassium",
    }
    var nutrientList []map[string]interface{}
    titleCaser := cases.Title(language.English)
    for _, key := range mainNutrients {
        if val, ok := nutriments[key]; ok {
            amount, ok := val.(float64)
            if !ok || amount <= 0 {
                continue
            }
            unit := "g"
            if u, ok := nutriments[key+"_unit"].(string); ok {
                unit = u
            }
            nutrientList = append(nutrientList, map[string]interface{}{
                "name":   titleCaser.String(strings.ReplaceAll(key, "-", " ")),
                "amount": amount,
                "unit":   unit,
            })
        }
    }
    return nutrientList
}