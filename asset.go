package bot

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type SwapAsset struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type SwapAssetsRespData struct {
	Assets []SwapAsset `json:"assets"`
}

type SwapAssetsResp struct {
	Ts   int64              `json:"ts"`
	Data SwapAssetsRespData `json:"data"`
}

func initAssets() (map[string]string, error) {
	resp, err := http.Get("https://api.4swap.org/api/assets")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result SwapAssetsResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	supportedAssets := make(map[string]string)
	for _, asset := range result.Data.Assets {
		supportedAssets[strings.ToUpper(asset.Symbol)] = asset.ID
	}

	return supportedAssets, nil
}
