package datapuller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPPuller HTTP REST API数据拉取器
type HTTPPuller struct {
	config *DataSourceConfig
	client *http.Client
}

// NewHTTPPuller 创建HTTP数据拉取器
func NewHTTPPuller(config *DataSourceConfig) *HTTPPuller {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &HTTPPuller{
		config: config,
		client: client,
	}
}

// PullBatch 拉取批量数据（非实时）
func (hp *HTTPPuller) PullBatch(ctx context.Context, start, end time.Time) ([]interface{}, error) {
	// 构建请求URL，包含时间范围参数
	url := fmt.Sprintf("%s?start_time=%d&end_time=%d",
		hp.config.URL,
		start.Unix(),
		end.Unix())

	var allData []interface{}
	page := 1
	limit := 100 // 每页100条记录

	for {
		// 添加分页参数
		paginatedURL := fmt.Sprintf("%s&page=%d&limit=%d", url, page, limit)

		req, err := http.NewRequestWithContext(ctx, "GET", paginatedURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// 添加API密钥到请求头
		if hp.config.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+hp.config.APIKey)
			req.Header.Set("X-API-Key", hp.config.APIKey)
		}

		req.Header.Set("Content-Type", "application/json")

		// 发送请求
		resp, err := hp.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}

		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		// 解析响应数据
		var pageData []interface{}
		if err := json.Unmarshal(body, &pageData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		// 如果没有更多数据，退出循环
		if len(pageData) == 0 {
			break
		}

		allData = append(allData, pageData...)

		// 如果返回的数据少于限制数量，说明已经是最后一页
		if len(pageData) < limit {
			break
		}

		page++

		// 添加小延迟以避免过于频繁的请求
		time.Sleep(100 * time.Millisecond)
	}

	return allData, nil
}

// PullLatest 拉取最新数据（非实时）
func (hp *HTTPPuller) PullLatest(ctx context.Context) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", hp.config.URL+"/latest", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 添加API密钥到请求头
	if hp.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+hp.config.APIKey)
		req.Header.Set("X-API-Key", hp.config.APIKey)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := hp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析响应数据
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return data, nil
}

// PullWithFilters 使用过滤器拉取数据
func (hp *HTTPPuller) PullWithFilters(ctx context.Context, filters map[string]interface{}) ([]interface{}, error) {
	// 构建带过滤参数的URL
	url := hp.config.URL
	queryParams := []string{}

	for key, value := range filters {
		queryParams = append(queryParams, fmt.Sprintf("%s=%v", key, value))
	}

	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 添加API密钥到请求头
	if hp.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+hp.config.APIKey)
		req.Header.Set("X-API-Key", hp.config.APIKey)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := hp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析响应数据
	var data []interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return data, nil
}

// PullRealTime 拉取实时数据
func (hp *HTTPPuller) PullRealTime(ctx context.Context, handler func(interface{}) error) error {
	// HTTP doesn't support true real-time, but we can simulate it with polling
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			data, err := hp.PullLatest(ctx)
			if err != nil {
				// Log error but continue polling
				fmt.Printf("Error pulling latest data: %v\n", err)
				continue
			}

			if err := handler(data); err != nil {
				// Log error but continue polling
				fmt.Printf("Error handling data: %v\n", err)
			}
		}
	}
}

// PullRealTimeEvents 实时拉取事件数据
func (hp *HTTPPuller) PullRealTimeEvents(ctx context.Context, handler func(interface{}) error) error {
	// For HTTP puller, PullRealTimeEvents behaves the same as PullRealTime
	return hp.PullRealTime(ctx, handler)
}

// PullHistorical 拉取历史数据
func (hp *HTTPPuller) PullHistorical(ctx context.Context, start, end time.Time, filters map[string]interface{}) ([]interface{}, error) {
	// First pull batch data by time range
	batchData, err := hp.PullBatch(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// Apply filters to the batch data if any filters are provided
	if len(filters) > 0 {
		filteredData := make([]interface{}, 0)
		for _, item := range batchData {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			match := true
			for key, value := range filters {
				itemValue, exists := itemMap[key]
				if !exists || fmt.Sprintf("%v", itemValue) != fmt.Sprintf("%v", value) {
					match = false
					break
				}
			}

			if match {
				filteredData = append(filteredData, item)
			}
		}

		return filteredData, nil
	}

	return batchData, nil
}

// Close 关闭HTTP拉取器
func (hp *HTTPPuller) Close() error {
	// HTTP客户端不需要特殊关闭，但我们可以重置它
	hp.client = nil
	return nil
}
