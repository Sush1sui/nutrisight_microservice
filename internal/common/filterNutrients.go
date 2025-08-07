package common

type Nutrient struct {
	NutrientName string  `json:"nutrientName"`
	Value        float64 `json:"value"`
	UnitName     string  `json:"unitName"`
}

func FilterNutrients(nutrients []Nutrient) []map[string]any {
	var filtered []map[string]any
	for _, n := range nutrients {
		if n.Value >= 0.1 {
			filtered = append(filtered, map[string]any{
				"name":   n.NutrientName,
				"amount": n.Value,
				"unit":   n.UnitName,
			})
		}
	}
	return filtered
}