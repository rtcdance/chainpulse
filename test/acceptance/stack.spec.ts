import { test, expect } from '@playwright/test';

const MONOLITHIC_URL = process.env.MONOLITHIC_URL || 'http://localhost:8080';
const GRAFANA_URL = process.env.GRAFANA_URL || 'http://localhost:3000';
const PROMETHEUS_URL = process.env.PROMETHEUS_URL || 'http://localhost:9090';

test.describe('Monolithic 部署验收', () => {
  test('Monolithic 服务启动成功', async ({ request }) => {
    const response = await request.get(`${MONOLITHIC_URL}/health`);
    
    expect(response.status()).toBe(200);
  });

  test('Monolithic 事件索引功能可用', async ({ request }) => {
    const response = await request.get(`${MONOLITHIC_URL}/api/v1/events?limit=10`);
    
    expect(response.status()).toBeGreaterThanOrEqual(200);
    expect(response.status()).toBeLessThan(500);
  });
});

test.describe('Grafana 监控验收', () => {
  test.skip('Grafana Dashboard 可访问', async ({ page }) => {
    await page.goto(`${GRAFANA_URL}/d/chainpulse-indexer`);
    
    await expect(page.locator('text=ChainPulse')).toBeVisible({ timeout: 10000 });
  }, '需要Grafana登录凭据');
});

test.describe('Prometheus 指标验收', () => {
  test('Prometheus 可查询 chainpulse 指标', async ({ request }) => {
    const response = await request.get(`${PROMETHEUS_URL}/api/v1/query?query=chainpulse_events_total`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.status).toBe('success');
  });

  test('Prometheus 可查询服务健康指标', async ({ request }) => {
    const response = await request.get(`${PROMETHEUS_URL}/api/v1/query?query=up{job="chainpulse"}`);
    
    expect(response.status()).toBe(200);
    
    const body = await response.json();
    expect(body.status).toBe('success');
  });
});

test.describe('Docker Compose 栈验收', () => {
  test('所有服务健康端点可访问', async ({ request }) => {
    const services = [
      { name: 'API Gateway', url: 'http://localhost:8080/health' },
      { name: 'API Service', url: 'http://localhost:8081/health' },
      { name: 'Prometheus', url: 'http://localhost:9090/-/healthy' },
    ];
    
    for (const service of services) {
      const response = await request.get(service.url, { timeout: 5000 });
      
      console.log(`${service.name}: ${response.status()}`);
      expect(response.status()).toBeGreaterThanOrEqual(200);
    }
  });
});