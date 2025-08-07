package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

    appkey := r.Header.Get("X-APP-KEY")
    if appkey != config.Global.SUSHI_SECRET_KEY {
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

func FoodScanHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    appkey := r.Header.Get("X-APP-KEY")
    if appkey != config.Global.SUSHI_SECRET_KEY {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Image string `json:"image"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Image == "" {
        http.Error(w, "No image provided", http.StatusBadRequest)
        return
    }

    // decode base64 image
    imgBytes, err := common.Base64ToBytes(req.Image)
    if err != nil {
        http.Error(w, "Invalid image format", http.StatusBadRequest)
        return
    }

    // send image to HuggingFace API
    hfReq, _ := http.NewRequest("POST", "https://api-inference.huggingface.co/models/nateraw/food", bytes.NewReader(imgBytes))
    hfReq.Header.Set("Authorization", "Bearer "+config.Global.HUGGINGFACE_API_KEY)
    hfReq.Header.Set("Content-Type", "application/octet-stream")
    hfResp, err := http.DefaultClient.Do(hfReq)
    if err != nil || hfResp.StatusCode != 200 {
        body, _ := io.ReadAll(hfResp.Body)
        http.Error(w, "Failed to fetch data from Hugging Face: "+string(body), http.StatusInternalServerError)
        return
    }
    defer hfResp.Body.Close()

    var predictions []struct {
        Label string  `json:"label"`
        Score float64 `json:"score"`
    }
    if err := json.NewDecoder(hfResp.Body).Decode(&predictions); err != nil || len(predictions) == 0 || predictions[0].Score < 0.5 {
        http.Error(w, "No food items detected in the image", http.StatusNotFound)
        return
    }

    // query USDA API with predicted label
     usdaURL := fmt.Sprintf(
        "https://api.nal.usda.gov/fdc/v1/foods/search?query=%s&dataType=Survey (FNDDS),Branded&api_key=%s",
        url.QueryEscape(predictions[0].Label),
        config.Global.USDA_API_KEY,
    )
    usdaResp, err := http.Get(usdaURL)
    if err != nil || usdaResp.StatusCode != 200 {
        http.Error(w, "Failed to fetch data from USDA API", http.StatusInternalServerError)
        return
    }
    defer usdaResp.Body.Close()

    var usdaData struct {
        Foods []struct {
            DataType      string `json:"dataType"`
            Description   string `json:"description"`
            Ingredients   string `json:"ingredients"`
            ServingSize   float64 `json:"servingSize"`
            ServingSizeUnit string `json:"servingSizeUnit"`
            PackageWeight string `json:"packageWeight"`
            FoodNutrients []struct {
                NutrientName string  `json:"nutrientName"`
                Value        float64 `json:"value"`
                UnitName     string  `json:"unitName"`
            } `json:"foodNutrients"`
        } `json:"foods"`
    }
    json.NewDecoder(usdaResp.Body).Decode(&usdaData)

    results := map[string]interface{}{
        "foodName": predictions[0].Label,
    }

    // get nutrition from first Survey (FNDDS) food
    for _, f := range usdaData.Foods {
        if f.DataType == "Survey (FNDDS)" {
            var nutrients []common.Nutrient
            for _, n := range f.FoodNutrients {
                if n.Value >= 0.1 {
                    nutrients = append(nutrients, common.Nutrient{
                        NutrientName: n.NutrientName,
                        Value:        n.Value,
                        UnitName:     n.UnitName,
                    })
                }
            }
            nutrition := common.ChunkArray(common.RenameNutrition(common.FilterNutrients(nutrients)), 6)
            for i := range nutrition {
                // flatten the chunked array to []map[string]any
                chunked := common.ChunkArray(nutrition[i], 2)
                flat := []map[string]any{}
                for _, arr := range chunked {
                    flat = append(flat, arr...)
                }
                nutrition[i] = flat
            }
            results["nutrition"] = nutrition
            break
        }
    }
    // get ingredients and serving size from first Branded food
    for _, f := range usdaData.Foods {
        if f.DataType == "Branded" && (f.PackageWeight != "" || (f.ServingSize > 0 && f.ServingSizeUnit != "")) && f.Ingredients != "" {
            results["ingredients"] = f.Ingredients
            if f.PackageWeight != "" {
                results["servingSize"] = f.PackageWeight
            } else {
                results["servingSize"] = fmt.Sprintf("%v%v", f.ServingSize, f.ServingSizeUnit)
            }
            break
        }
    }

    resp := map[string]interface{}{
        "message": "Food scan data received successfully",
        "data":    results,
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)

}