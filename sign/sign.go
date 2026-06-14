package sign

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://www.natpierce.cn"

var commonHeaders = map[string]string{
	"Content-Type":     "application/x-www-form-urlencoded",
	"X-Requested-With": "XMLHttpRequest",
	"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Accept":           "application/json, text/javascript, */*; q=0.01",
}

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Client is an HTTP client for signing in to natpierce.cn
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new sign-in client
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, apiURL string, body []byte, referer string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Login logs in to the website
func (c *Client) Login(username, password string) error {
	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)

	body, err := c.doRequest("POST", baseURL+"/pc/login/login.html", []byte(form.Encode()), baseURL+"/pc/login/login.html")
	if err != nil {
		return fmt.Errorf("登录请求失败: %w", err)
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w (响应: %s)", err, string(body))
	}

	if result.Code != 200 {
		return fmt.Errorf("%s", result.Message)
	}
	return nil
}

// PreCheck checks if sign-in is needed
func (c *Client) PreCheck() (bool, string, error) {
	body, err := c.doRequest("POST", baseURL+"/pc/sign/qiandao_bf.html", nil, baseURL+"/pc/sign/index.html")
	if err != nil {
		return false, "", fmt.Errorf("签到检查请求失败: %w", err)
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, "", fmt.Errorf("解析响应失败: %w (响应: %s)", err, string(body))
	}

	return result.Code == 200, result.Message, nil
}

// Sign performs the sign-in
func (c *Client) Sign() (bool, string, error) {
	body, err := c.doRequest("POST", baseURL+"/pc/sign/qiandao.html", nil, baseURL+"/pc/sign/index.html")
	if err != nil {
		return false, "", fmt.Errorf("签到请求失败: %w", err)
	}

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, "", fmt.Errorf("解析响应失败: %w (响应: %s)", err, string(body))
	}

	return result.Code == 200, result.Message, nil
}

// SignInAccount performs sign-in for a single account, returns log messages
func SignInAccount(username, password string) []string {
	var logs []string
	client := NewClient()

	logs = append(logs, fmt.Sprintf("[%s] 正在登录...", username))
	if err := client.Login(username, password); err != nil {
		logs = append(logs, fmt.Sprintf("[%s] ❌ 登录失败: %v", username, err))
		return logs
	}
	logs = append(logs, fmt.Sprintf("[%s] ✅ 登录成功", username))

	logs = append(logs, fmt.Sprintf("[%s] 检查签到状态...", username))
	canSign, msg, err := client.PreCheck()
	if err != nil {
		logs = append(logs, fmt.Sprintf("[%s] ❌ 签到检查失败: %v", username, err))
		return logs
	}

	if !canSign {
		logs = append(logs, fmt.Sprintf("[%s] ℹ️ 无需签到: %s", username, msg))
		return logs
	}
	logs = append(logs, fmt.Sprintf("[%s] 可以签到: %s", username, msg))

	logs = append(logs, fmt.Sprintf("[%s] 正在签到...", username))
	ok, signMsg, err := client.Sign()
	if err != nil {
		logs = append(logs, fmt.Sprintf("[%s] ❌ 签到失败: %v", username, err))
		return logs
	}

	if ok {
		logs = append(logs, fmt.Sprintf("[%s] ✅ 签到成功: %s", username, signMsg))
	} else {
		logs = append(logs, fmt.Sprintf("[%s] ❌ 签到失败: %s", username, signMsg))
	}
	return logs
}
