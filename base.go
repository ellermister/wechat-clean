package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// 并集
func union(slice1, slice2 []string) []string {
	m := make(map[string]int)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 0 {
			slice1 = append(slice1, v)
		}
	}
	return slice1
}

// 交集
func intersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 1 {
			nn = append(nn, v)
		}
	}
	return nn
}

func difference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := intersect(slice1, slice2)
	for _, v := range inter {
		m[v]++
	}

	for _, value := range slice1 {
		times, _ := m[value]
		if times == 0 {
			nn = append(nn, value)
		}
	}
	return nn
}

func removeDuplicate(arr []string) []string {
	result := make([]string, 0, len(arr))
	temp := map[string]struct{}{}
	for _, item := range arr {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return true
	}
	return true
}

func chunksIds(ids []int64, chunkSize int) [][]int64 {
	var data [][]int64
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize

		if end > len(ids) {
			end = len(ids)
		}

		data = append(data, ids[i:end])
	}
	return data
}

func implodeI2S(ids []int64, char string) string {
	var strIds []string
	for _, id := range ids {
		strIds = append(strIds, strconv.FormatInt(id, 10))
	}
	return strings.Join(strIds, char)
}

func DisableAPP(packageName string) {
	cmd := exec.Command("pm", "disable", packageName)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to Disable app:%s, err: %e", packageName, err)
	}
}

func EnableApp(packageName string) {
	cmd := exec.Command("pm", "enable", packageName)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to Enable app:%s, err: %e", packageName, err)
	}
}

type ServerResponseJson struct {
	Command string `json:"command"`
	Status  int    `json:"status"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func rjson(command string, message string, status int, data any) []byte {
	var response ServerResponseJson
	response.Data = data
	response.Status = status
	response.Command = command
	response.Message = message
	jsonString, err := json.Marshal(response)
	if err != nil {
		log.Printf("json.Marshal err: %s", err)
		return []byte{}
	} else {
		return jsonString
	}
}
