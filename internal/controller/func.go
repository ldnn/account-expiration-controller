package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sha256Hash(data string) string {
	// 将字符串转换为字节数组
	dataBytes := []byte(data)

	// 使用 SHA-256 创建哈希对象
	hash := sha256.New()

	// 将数据添加到哈希对象中
	hash.Write(dataBytes)

	// 计算 SHA-256 值
	sha256Value := hash.Sum(nil)

	// 将 SHA-256 值转换为十六进制字符串
	sha256Hex := hex.EncodeToString(sha256Value)

	return sha256Hex
}

func CheckStatus(appId string, appSecret string, api string, phoneNubmber string) bool {

	unixTime := time.Now().Unix()
	timestamp := strconv.FormatInt(unixTime, 10)

	hash_str := fmt.Sprintf("%v%v%v", appId, appSecret, timestamp)
	token := sha256Hash(hash_str)

	when := fmt.Sprintf("%v\"%v\"", "phoneNumbers eq ", phoneNubmber)
	//fmt.Print(when)

	// 创建 URL 对象
	u, err := url.Parse(api)
	if err != nil {
		log.Printf("Failed to parse URL: %v", err)
		return true
	}

	// 创建查询参数
	params := url.Values{}
	params.Add("filter", when)

	u.RawQuery = params.Encode()

	// 创建 GET 请求
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return true
	}

	// 设置自定义头部
	req.Header.Set("X-App-Token", token)
	req.Header.Set("X-App-Id", appId)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("Content-Type", "application/json")

	// 定义超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// 确保函数返回时取消上下文
	defer cancel()

	// 将 context 关联到请求
	req = req.WithContext(ctx)

	// 创建 HTTP 客户端
	client := &http.Client{}

	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return true
	}
	defer resp.Body.Close()

	// 打印响应状态
	if resp.StatusCode != 200 {
		log.Printf("请求失败！！！")
		return true
	}
	//fmt.Println("Response status:", resp.Status)

	// 读取响应体

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体出错: %v", err)
		return true
	}

	if len(body) == 0 {
		log.Printf("响应体为空.")
		return true
	}

	var content Content
	err = json.Unmarshal([]byte(body), &content)

	// 打印响应体
	if content.TotalResults == 0 {
		log.Printf("没有找到手机号对应的4A账号.")
		return true
	}
	log.Printf("%v: %v", phoneNubmber, content.Resources[0].Active)
	return content.Resources[0].Active
}

func CheckLock(uLastLoginTime *v1.Time, uLastTransitionTime *v1.Time, uCreationTimestamp time.Time) bool {
	switch {
	case uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
		if uCreationTimestamp.AddDate(0, 3, 0).Before(time.Now()) {
			return false
		}
	case !uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
		switch {
		case uLastTransitionTime.AddDate(0, 3, 0).Before(time.Now()) && uCreationTimestamp.Equal(uLastTransitionTime.Time):
			return false
		case uLastTransitionTime.AddDate(0, 0, 7).Before(time.Now()) && !uCreationTimestamp.Equal(uLastTransitionTime.Time):
			if uCreationTimestamp.AddDate(0, 3, 0).Before(time.Now()) {
				return false
			}
		}
	case uLastTransitionTime.IsZero() && !uLastLoginTime.IsZero():
		if uLastLoginTime.AddDate(0, 3, 0).Before(time.Now()) {
			return false
		}

	case uLastLoginTime.AddDate(0, 3, 0).Before(time.Now()) && uLastTransitionTime.AddDate(0, 0, 7).Before(time.Now()):
		return false
	}

	return true
}

func CheckDel(uLastLoginTime *v1.Time, uLastTransitionTime *v1.Time, uCreationTimestamp time.Time) bool {
	switch {
	case uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
		if uCreationTimestamp.AddDate(0, 6, 0).Before(time.Now()) {
			return false
		}
	case !uLastTransitionTime.IsZero() && uLastLoginTime.IsZero():
		if uLastTransitionTime.AddDate(0, 6, 0).Before(time.Now()) {
			return false
		}
	case uLastTransitionTime.IsZero() && !uLastLoginTime.IsZero():
		if uLastLoginTime.AddDate(0, 6, 0).Before(time.Now()) {
			return false
		}
	case uLastTransitionTime.AddDate(0, 6, 0).Before(time.Now()) && uLastLoginTime.AddDate(0, 6, 0).Before(time.Now()):
		return false
	}

	return true
}
