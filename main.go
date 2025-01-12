package main

import (
	"github.com/amankumarsingh77/go-showbox-api/scraper/showbox"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := showbox.DefaultConfig()
	storage := showbox.NewStorage()

	scraper, err := showbox.NewScraper(config, storage)
	if err != nil {
		log.Fatalf("Failed to create scraper: %v", err)
	}

	if err := scraper.Run(); err != nil {
		log.Fatalf("Scraper error: %v", err)
	}
}

//package main
//
//import (
//	"bytes"
//	"crypto/cipher"
//	"crypto/des"
//	"crypto/md5"
//	"encoding/base64"
//	"encoding/json"
//	"errors"
//	"fmt"
//	"io/ioutil"
//	"math/rand"
//	"net/http"
//	"net/url"
//	"strconv"
//	"strings"
//	"time"
//)
//
//const (
//	iv       = "wEiphTn!"
//	key      = "123d6cedf626dy54233aa1w6"
//	appKey   = "moviebox"
//	appId    = "com.tdo.showbox"
//	baseURL1 = "https://showbox.shegu.net/api/api_client/index/"
//	baseURL2 = "https://mbpapi.shegu.net/api/api_client/index/"
//)
//
//var alphabet = "0123456789abcdef"
//
//func nanoid(size int) string {
//	rand.Seed(time.Now().UnixNano())
//	nanoid := make([]byte, size)
//	for i := range nanoid {
//		nanoid[i] = alphabet[rand.Intn(len(alphabet))]
//	}
//	return string(nanoid)
//}
//
//func encrypt(data string) (string, error) {
//	block, err := des.NewTripleDESCipher([]byte(key))
//	if err != nil {
//		return "", err
//	}
//	ivBytes := []byte(iv)
//	mode := cipher.NewCBCEncrypter(block, ivBytes)
//	padded := pkcs5Padding([]byte(data), block.BlockSize())
//	encrypted := make([]byte, len(padded))
//	mode.CryptBlocks(encrypted, padded)
//	return base64.StdEncoding.EncodeToString(encrypted), nil
//}
//
//func pkcs5Padding(src []byte, blockSize int) []byte {
//	padding := blockSize - len(src)%blockSize
//	padText := bytes.Repeat([]byte{byte(padding)}, padding)
//	return append(src, padText...)
//}
//
//func getVerify(str, str2, str3 string) string {
//	if str != "" {
//		hash := md5.Sum([]byte(fmt.Sprintf("%x%s%s", md5.Sum([]byte(str2)), str3, str)))
//		return fmt.Sprintf("%x", hash)
//	}
//	return ""
//}
//
//func expiry() int64 {
//	return time.Now().Unix() + 60*60*12
//}
//
//func get(data map[string]interface{}, altAPI bool) (map[string]interface{}, error) {
//	defaultData := map[string]interface{}{
//		"childmode":    "0",
//		"app_version":  "11.5",
//		"appid":        appId,
//		"lang":         "en",
//		"expired_date": strconv.FormatInt(expiry(), 10),
//		"platform":     "android",
//		"channel":      "Website",
//	}
//
//	for k, v := range data {
//		defaultData[k] = v
//	}
//
//	dataBytes, err := json.Marshal(defaultData)
//	if err != nil {
//		return nil, err
//	}
//
//	encryptedData, err := encrypt(string(dataBytes))
//	if err != nil {
//		return nil, err
//	}
//
//	appKeyHash := fmt.Sprintf("%x", md5.Sum([]byte(appKey)))
//	verify := getVerify(encryptedData, appKey, key)
//	body := map[string]string{
//		"app_key":      appKeyHash,
//		"verify":       verify,
//		"encrypt_data": encryptedData,
//	}
//	bodyBytes, err := json.Marshal(body)
//	if err != nil {
//		return nil, err
//	}
//
//	b64Body := base64.StdEncoding.EncodeToString(bodyBytes)
//	form := url.Values{}
//	form.Add("data", b64Body)
//	form.Add("appid", "27")
//	form.Add("platform", "android")
//	form.Add("version", "129")
//	form.Add("medium", "Website")
//
//	requestURL := baseURL1
//	if altAPI {
//		requestURL = baseURL2
//	}
//
//	resp, err := http.Post(requestURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()+"&token"+nanoid(32)))
//
//	if err != nil {
//		return nil, err
//	}
//	defer resp.Body.Close()
//
//	bodyData, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		return nil, err
//	}
//
//	var response map[string]interface{}
//	err = json.Unmarshal(bodyData, &response)
//	if err != nil {
//		return nil, err
//	}
//
//	return response, nil
//}
//
//func compareTitle(title1, title2 string) bool {
//	normalizedTitle1 := strings.TrimSpace(strings.ToLower(title1))
//	normalizedTitle2 := strings.TrimSpace(strings.ToLower(title2))
//	return normalizedTitle1 == normalizedTitle2
//}
//
//func execute(titleInfo map[string]interface{}, setProgress func(float64)) ([]map[string]interface{}, error) {
//	// Construct the search query
//	searchQuery := map[string]interface{}{
//		"module":    "Search3",
//		"page":      "1",
//		"type":      "all",
//		"keyword":   titleInfo["title"],
//		"pagelimit": "20",
//	}
//
//	searchRes, err := get(searchQuery, true)
//	if err != nil {
//		return nil, err
//	}
//
//	searchData, ok := searchRes["data"].([]interface{})
//	if !ok || len(searchData) == 0 {
//		return nil, errors.New("No search results found")
//	}
//
//	setProgress(0.5)
//
//	// Find the matching entry
//	var superstreamEntry map[string]interface{}
//	for _, entry := range searchData {
//		item, _ := entry.(map[string]interface{})
//		if compareTitle(item["title"].(string), titleInfo["title"].(string)) &&
//			item["year"] == titleInfo["year"] {
//			superstreamEntry = item
//			break
//		}
//	}
//
//	if superstreamEntry == nil {
//		return nil, errors.New("No stream found")
//	}
//
//	superstreamId, ok := superstreamEntry["id"].(string)
//	if !ok {
//		return nil, errors.New("Invalid stream ID")
//	}
//
//	// Determine if the title is a movie or show
//	if titleInfo["type"] == "movie" {
//		// Fetch movie details
//		apiQuery := map[string]interface{}{
//			"uid":    "",
//			"module": "Movie_downloadurl_v3",
//			"mid":    superstreamId,
//			"oss":    "1",
//			"group":  "",
//		}
//
//		watchRes, err := get(apiQuery, false)
//		if err != nil {
//			return nil, err
//		}
//
//		watchInfo, ok := watchRes["data"].(map[string]interface{})
//		if !ok || watchInfo["list"] == nil {
//			return nil, errors.New("No stream found")
//		}
//
//		watchList, _ := watchInfo["list"].([]interface{})
//		var results []map[string]interface{}
//		for _, item := range watchList {
//			entry, _ := item.(map[string]interface{})
//			if entry["path"] != "" {
//				quality := entry["real_quality"]
//				parsedQuality := "unknown"
//				if quality == "4K" {
//					parsedQuality = "4k"
//				} else if intVal, err := strconv.Atoi(fmt.Sprintf("%v", quality)); err == nil {
//					parsedQuality = strconv.Itoa(intVal)
//				}
//				results = append(results, map[string]interface{}{
//					"quality": parsedQuality,
//					"url":     entry["path"],
//				})
//			}
//		}
//		return results, nil
//
//	} else if titleInfo["type"] == "show" {
//		// Fetch show details
//		apiQuery := map[string]interface{}{
//			"uid":     "",
//			"module":  "TV_downloadurl_v3",
//			"tid":     superstreamId,
//			"season":  titleInfo["season"],
//			"episode": titleInfo["episode"],
//			"oss":     "1",
//			"group":   "",
//		}
//
//		watchRes, err := get(apiQuery, false)
//		if err != nil {
//			return nil, err
//		}
//
//		watchInfo, ok := watchRes["data"].(map[string]interface{})
//		if !ok || watchInfo["list"] == nil {
//			return nil, errors.New("No stream found")
//		}
//
//		watchList, _ := watchInfo["list"].([]interface{})
//		var results []map[string]interface{}
//		for _, item := range watchList {
//			entry, _ := item.(map[string]interface{})
//			if entry["path"] != "" {
//				quality := entry["real_quality"]
//				parsedQuality := "unknown"
//				if quality == "4K" {
//					parsedQuality = "4k"
//				} else if intVal, err := strconv.Atoi(fmt.Sprintf("%v", quality)); err == nil {
//					parsedQuality = strconv.Itoa(intVal)
//				}
//				results = append(results, map[string]interface{}{
//					"quality": parsedQuality,
//					"url":     entry["path"],
//				})
//			}
//		}
//		return results, nil
//	}
//
//	return nil, errors.New("invalid media type")
//}
//
//// Mock function to set progress
//func setProgress(progress float64) {
//	fmt.Printf("Progress: %.2f%%\n", progress*100)
//}
//
//func main() {
//	// Example input
//	titleInfo := map[string]interface{}{
//		"title":   "avengers",
//		"type":    "movie", // or "show"
//		"year":    2023,
//		"season":  1, // Only required for shows
//		"episode": 1, // Only required for shows
//	}
//
//	// Execute function
//	results, err := execute(titleInfo, setProgress)
//	if err != nil {
//		fmt.Println("Error:", err)
//		return
//	}
//
//	// Display results
//	fmt.Println("Streaming Links:")
//	formattedResults, _ := json.MarshalIndent(results, "", "  ")
//	fmt.Println(string(formattedResults))
//}
