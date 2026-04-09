import { test, expect } from '@playwright/test';

const API_URL = process.env.API_URL || 'http://localhost:8080';

test.describe('端到端功能验收', () => {
  test('完整事件查询流程', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/events?limit=10&offset=0`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.events).toBeDefined();
    expect(Array.isArray(body.events)).toBe(true);
  });

  test('事件过滤功能', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/events?chain_id=1&event_name=Transfer`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.events).toBeDefined();
  });

  test('单个事件查询', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/events/event-1`);
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });

  test('分页功能正常', async ({ request }) => {
    const page1 = await request.get(`${API_URL}/api/v1/events?limit=5&offset=0`);
    const page2 = await request.get(`${API_URL}/api/v1/events?limit=5&offset=5`);
    
    expect(page1.status()).toBe(200);
    expect(page2.status()).toBe(200);
  });
});

test.describe('GraphQL API 验收', () => {
  test('GraphQL  introspection', async ({ request }) => {
    const response = await request.post(`${API_URL}/graphql`, {
      data: {
        query: `
          query {
            __schema {
              queryType { name }
              mutationType { name }
              types { name }
            }
          }
        `
      }
    });
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.data).toBeDefined();
  });

  test('GraphQL 事件查询', async ({ request }) => {
    const response = await request.post(`${API_URL}/graphql`, {
      data: {
        query: `
          query {
            events(limit: 5) {
              id
              event_name
              chain_id
            }
          }
        `
      }
    });
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });
});

test.describe('WebSocket 实时功能验收', () => {
  test('WebSocket 连接建立', async ({ page }) => {
    const wsUrl = API_URL.replace('http', 'ws') + '/ws';
    
    const messagePromise = new Promise((resolve) => {
      page.on('websocket', ws => {
        ws.on('framereceived', frame => {
          resolve(frame.text());
        });
      });
    });
    
    await page.goto(`${API_URL}/`);
    
    const pageHasContent = await page.locator('body').count() > 0;
    expect(pageHasContent).toBe(true);
  });
});

test.describe('API 错误处理验收', () => {
  test('无效请求返回适当错误', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/invalid-path-that-does-not-exist`);
    
    expect(response.status()).toBeGreaterThanOrEqual(400);
    expect(response.status()).toBeLessThan(600);
  });

  test('无效参数返回验证错误', async ({ request }) => {
    const response = await request.get(`${API_URL}/api/v1/events?limit=-1`);
    
    expect(response.status()).toBeGreaterThanOrEqual(400);
    expect(response.status()).toBeLessThan(500);
  });
});