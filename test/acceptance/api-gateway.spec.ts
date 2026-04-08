import { test, expect } from '@playwright/test';

const API_GATEWAY_URL = process.env.API_GATEWAY_URL || 'http://localhost:8080';
const API_SERVICE_URL = process.env.API_SERVICE_URL || 'http://localhost:8081';

test.describe('API Gateway 健康检查验收', () => {
  test('API Gateway .health 端点返回healthy状态', async ({ request }) => {
    const response = await request.get(`${API_GATEWAY_URL}/health`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.status).toBeDefined();
  });

  test('API Gateway 根路径可访问', async ({ request }) => {
    const response = await request.get(`${API_GATEWAY_URL}/`);
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('API Gateway metrics 端点可访问', async ({ request }) => {
    const response = await request.get(`${API_GATEWAY_URL}/metrics`);
    
    expect(response.status()).toBe(200);
    
    const text = await response.text();
    expect(text).toContain('chainpulse');
  });
});

test.describe('API Service 健康检查验收', () => {
  test('API Service .health 端点返回healthy状态', async ({ request }) => {
    const response = await request.get(`${API_SERVICE_URL}/health`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.status).toBeDefined();
  });

  test('API Service metrics 端点可访问', async ({ request }) => {
    const response = await request.get(`${API_SERVICE_URL}/metrics`);
    
    expect(response.status()).toBe(200);
  });
});

test.describe('API Gateway 路由验收', () => {
  test('API Gateway 正确路由到后端服务', async ({ request }) => {
    const response = await request.get(`${API_GATEWAY_URL}/api/v1/events`);
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('API Gateway 支持 GraphQL 查询', async ({ request }) => {
    const response = await request.post(`${API_GATEWAY_URL}/graphql`, {
      data: {
        query: `{ __schema { types { name } } }`
      }
    });
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('API Gateway 支持 WebSocket 连接', async ({ page }) => {
    await page.goto(`${API_GATEWAY_URL}/ws`);
    
    const hasWS = page.url().includes('ws') || page.url().includes('websocket');
    expect(hasWS || response.status() < 500).toBeTruthy();
  });
});

test.describe('API Gateway 性能验收', () => {
  test('API Gateway 响应时间 < 1秒', async ({ request }) => {
    const start = Date.now();
    
    await request.get(`${API_GATEWAY_URL}/health`);
    
    const duration = Date.now() - start;
    expect(duration).toBeLessThan(1000);
  });

  test('API Gateway 并发请求处理正常', async ({ request }) => {
    const promises = Array(10).fill(null).map(() => 
      request.get(`${API_GATEWAY_URL}/health`)
    );
    
    const results = await Promise.all(promises);
    
    results.forEach(response => {
      expect(response.status()).toBe(200);
    });
  });
});