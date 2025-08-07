package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Sush1sui/internal/common"
	"github.com/Sush1sui/internal/config"
)



func IndexHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the NutriSight API!"))
}

func BarcodeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if r.Header.Get("X-APP-KEY") != os.Getenv("SUSHI_SECRET_KEY") {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

	var req struct {
		BarcodeData string `json:"barcodeData"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BarcodeData == "" {
        http.Error(w, "No barcode data provided", http.StatusBadRequest)
        return
    }

	// USDA API
	usdaURL := fmt.Sprintf("https://api.nal.usda.gov/fdc/v1/foods/search?query=%s", req.BarcodeData)
    usdaReq, _ := http.NewRequest("GET", usdaURL, nil)
    usdaReq.Header.Set("x-api-key", config.Global.USDA_API_KEY)
    usdaResp, err := http.DefaultClient.Do(usdaReq)
	if err == nil && usdaResp.StatusCode == 200 {
		defer usdaResp.Body.Close()
		var usdaData struct {
            Foods []struct {
                Description   string `json:"description"`
                BrandOwner    string `json:"brandOwner"`
                Ingredients   string `json:"ingredients"`
                ServingSize   float64 `json:"servingSize"`
                ServingSizeUnit string `json:"servingSizeUnit"`
                FoodNutrients []struct {
                    NutrientName string  `json:"nutrientName"`
                    Value        float64 `json:"value"`
                    UnitName     string  `json:"unitName"`
                } `json:"foodNutrients"`
            } `json:"foods"`
        }
		json.NewDecoder(usdaResp.Body).Decode(&usdaData)
		if len(usdaData.Foods) > 0 {
			food := usdaData.Foods[0]
			var nutrients []common.Nutrient
			for _, n := range food.FoodNutrients {
				nutrients = append(nutrients, common.Nutrient{
					NutrientName: n.NutrientName,
					Value:        n.Value,
					UnitName:     n.UnitName,
				})
			}
			nutrition := common.ChunkArray(common.RenameNutrition(common.FilterNutrients(nutrients)), 6)
			resp := map[string]interface{}{
                "message": "Barcode data received successfully",
                "data": map[string]interface{}{
                    "name":        food.Description,
                    "brand":       food.BrandOwner,
                    "ingredients": food.Ingredients,
                    "nutrition":   nutrition,
                    "servingSize": fmt.Sprintf("%v%v", food.ServingSize, food.ServingSizeUnit),
                },
            }
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Nutritionix Fallback
	nutriURL := fmt.Sprintf("https://trackapi.nutritionix.com/v2/search/item?upc=%s", req.BarcodeData)
	nutriReq, _ := http.NewRequest("GET", nutriURL, nil)
	nutriReq.Header.Set("x-app-id", config.Global.NUTRITIONIX_APP_ID)
    nutriReq.Header.Set("x-app-key", config.Global.NUTRITIONIX_API_KEY)
    nutriResp, err := http.DefaultClient.Do(nutriReq)
	if err == nil && nutriResp.StatusCode == 200 {
		defer nutriResp.Body.Close()
		var nutriData struct {
            Foods []struct {
                FoodName             string `json:"food_name"`
                BrandName            string `json:"brand_name"`
                NfIngredientStatement string `json:"nf_ingredient_statement"`
				ServingQty            float64 `json:"serving_qty"`
        		ServingUnit           string  `json:"serving_unit"`
        		ServingWeightGrams    float64 `json:"serving_weight_grams"`
                FullNutrients        []struct {
                    AttrID int     `json:"attr_id"`
                    Value  float64 `json:"value"`
                } `json:"full_nutrients"`
            } `json:"foods"`
        }
		json.NewDecoder(nutriResp.Body).Decode(&nutriData)
		if len(nutriData.Foods) > 0 {
			food := nutriData.Foods[0]
			nutrientMap := map[int]struct {
                Name string
                Unit string
            }{
                203: {"Protein", "g"},
                204: {"Total lipid (fat)", "g"},
                205: {"Carbohydrate, by difference", "g"},
                208: {"Energy", "kcal"},
                269: {"Total Sugars", "g"},
                291: {"Fiber, total dietary", "g"},
                301: {"Calcium, Ca", "mg"},
                303: {"Iron, Fe", "mg"},
                307: {"Sodium, Na", "mg"},
                318: {"Vitamin A, IU", "IU"},
                401: {"Vitamin C, total ascorbic acid", "mg"},
                601: {"Cholesterol", "mg"},
                605: {"Fatty acids, total trans", "g"},
                606: {"Fatty acids, total saturated", "g"},
            }
			var nutritionData []map[string]any
			for _, n := range food.FullNutrients {
				if info, ok := nutrientMap[n.AttrID]; ok {
					nutritionData = append(nutritionData, map[string]any{
						"name":   info.Name,
						"amount": n.Value,
						"unit":   info.Unit,
					})
				}
			}

			// filter and chunk
			filtered := []map[string]any{}
			for _, n := range nutritionData {
                if val, ok := n["amount"].(float64); ok && val >= 0.1 {
                    filtered = append(filtered, n)
                }
            }
			servingSize := "N/A"
			nutrition := common.ChunkArray(common.RenameNutrition(filtered), 6)
			if food.ServingQty > 0 && food.ServingUnit != "" && food.ServingWeightGrams > 0 {
				servingSize = fmt.Sprintf("%v %v (%.0fg)", food.ServingQty, food.ServingUnit, food.ServingWeightGrams)
			}
			resp := map[string]interface{}{
                "message": "Barcode data received successfully from Nutritionix",
                "data": map[string]interface{}{
                    "name":        food.FoodName,
                    "brand":       food.BrandName,
                    "ingredients": food.NfIngredientStatement,
                    "nutrition":   nutrition,
                    "servingSize": servingSize,
                },
            }
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// If both APIs fail
	// Use open food facts as a last resort
	offURL := fmt.Sprintf("https://world.openfoodfacts.net/api/v2/product/%s.json", req.BarcodeData)
    offReq, _ := http.NewRequest("GET", offURL, nil)
    offReq.Header.Set("User-Agent", "nutrisight-thesis/1.0 - (github.com/Sush1sui)")
    offResp, err := http.DefaultClient.Do(offReq)
    if err != nil || offResp.StatusCode != 200 {
        http.Error(w, "Failed to fetch data.", http.StatusInternalServerError)
        return
    }
    defer offResp.Body.Close()
    var offData struct {
        Product struct {
            ProductName    string                 `json:"product_name"`
            Brands         string                 `json:"brands"`
            IngredientsText string                `json:"ingredients_text"`
            Nutriments     map[string]interface{} `json:"nutriments"`
            ServingSize    string                 `json:"serving_size"`
        } `json:"product"`
    }
    json.NewDecoder(offResp.Body).Decode(&offData)
    if offData.Product.ProductName == "" {
        http.Error(w, "No product found for the barcode", http.StatusNotFound)
        return
    }
    nutrition := common.ChunkArray(common.FormatNutriments(offData.Product.Nutriments), 6)
    resp := map[string]interface{}{
        "message": "Barcode data received successfully from Open Food Facts",
        "data": map[string]interface{}{
            "name":        offData.Product.ProductName,
            "brand":       offData.Product.Brands,
            "ingredients": offData.Product.IngredientsText,
            "nutrition":   nutrition,
            "servingSize": offData.Product.ServingSize,
        },
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}