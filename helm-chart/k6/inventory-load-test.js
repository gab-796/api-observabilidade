import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';

/***************************************
 * Configuração básica - TESTE LEVE
 * - Max 10 VUs
 * - Max 10 requisições/segundo
 * - Ramp up de 3 minutos
 ***************************************/
export const options = {
  scenarios: {
    light_load: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: 2,
      maxVUs: 10,
      stages: [
        { duration: '30s', target: 5 },   // ramp up suave - 5 req/s
        { duration: '60s', target: 10 },  // ramp up para 10 req/s
        { duration: '60s', target: 10 },  // mantém 10 req/s
        { duration: '30s', target: 0 },   // ramp down
      ],
      exec: 'mainScenario'
    }
  },
  thresholds: {
    http_req_duration: ['p(95)<800', 'p(99)<1500'],
    http_req_failed: ['rate<0.05'],  // tolerância maior para teste leve
    'inventory_requests_total{endpoint:createProduct}': ['count>0'],
  }
};

/***************************************
 * Métricas customizadas
 ***************************************/
const createProductTrend = new Trend('create_product_duration');
const listProductsTrend = new Trend('list_products_duration');
const getProductTrend = new Trend('get_product_duration');
const deleteProductTrend = new Trend('delete_product_duration');
const businessErrorRate = new Rate('business_errors');
const inventoryRequests = new Counter('inventory_requests_total');

/***************************************
 * Novas métricas para rotas adicionais
 ***************************************/
const getSingleProductTrend = new Trend('get_single_product_duration');
const updateProductTrend = new Trend('update_product_duration');

/***************************************
 * Helpers
 ***************************************/
const BASE_URL = __ENV.BASE_URL || 'http://inventory.local';
const METRICS_ENABLED = (__ENV.METRICS_ENABLED || 'true') === 'true';

function randomString(len = 6) {
  const chars = 'abcdefghijklmnopqrstuvwxyz';
  let s = '';
  for (let i = 0; i < len; i++) s += chars[Math.floor(Math.random()*chars.length)];
  return s;
}

function randomPrice() { return (Math.random() * 100).toFixed(2); }

/***************************************
 * Fluxos de requisições
 ***************************************/
export function mainScenario() {
  group('inventory-crud-flow', () => {
    // Listar produtos (GET /products)
    let resList = http.get(`${BASE_URL}/products`);
    listProductsTrend.add(resList.timings.duration);
    check(resList, { 'list status 200': r => r.status === 200 }) || businessErrorRate.add(1);
    inventoryRequests.add(1, { endpoint: 'listProducts' });

    // Criar produto (POST /product) – algumas versões usam /product em vez de /products para criação
    const newProduct = {
      name: `prod-${randomString()}`,
      price: parseFloat(randomPrice()),
      quantity: Math.floor(Math.random() * 50) + 1
    };
    let resCreate = http.post(`${BASE_URL}/product`, JSON.stringify(newProduct), { headers: { 'Content-Type': 'application/json' } });
    createProductTrend.add(resCreate.timings.duration);
    const createdOk = check(resCreate, {
      'create 201|200': r => r.status === 201 || r.status === 200,
      'create has id': r => { try { return JSON.parse(r.body).id !== undefined; } catch { return false; } }
    });
    if (!createdOk) businessErrorRate.add(1);
    inventoryRequests.add(1, { endpoint: 'createProduct' });

    let productId = null;
    try { productId = JSON.parse(resCreate.body).id; } catch { /* ignore */ }

    // GET /product/:id
    if (productId) {
      let resGetSingle = http.get(`${BASE_URL}/product/${productId}`);
      getSingleProductTrend.add(resGetSingle.timings.duration);
      check(resGetSingle, {
        'get single 200': r => r.status === 200,
        'get single id ok': r => { try { return JSON.parse(r.body).id === productId; } catch { return false; } }
      }) || businessErrorRate.add(1);
      inventoryRequests.add(1, { endpoint: 'getProduct' });
    }

    // PUT /product/:id (atualiza)
    if (productId) {
      const updateBody = { name: `prod-upd-${randomString()}`, price: parseFloat(randomPrice()), quantity: Math.floor(Math.random() * 50) + 1 };
      const resUpdate = http.put(`${BASE_URL}/product/${productId}`, JSON.stringify(updateBody), { headers: { 'Content-Type': 'application/json' } });
      updateProductTrend.add(resUpdate.timings.duration);
      check(resUpdate, { 'update 200': r => r.status === 200 }) || businessErrorRate.add(1);
      inventoryRequests.add(1, { endpoint: 'updateProduct' });
    }

    // DELETE /product/:id (70% dos casos)
    if (productId && Math.random() < 0.7) {
      let resDel = http.del(`${BASE_URL}/product/${productId}`);
      deleteProductTrend.add(resDel.timings.duration);
      check(resDel, { 'delete 200|204|202': r => [200,204,202].includes(r.status) }) || businessErrorRate.add(1);
      inventoryRequests.add(1, { endpoint: 'deleteProduct' });
    }
  });

  // Sleep mais longo para controlar a taxa de requisições
  sleep(Math.random() * 2 + 1); // Entre 1-3 segundos
}

/***************************************
 * Execução isolada opcional (k6 run --vus 1 --duration 10s inventory-load-test.js)
 ***************************************/
export function setup() {
  console.log(`Base URL: ${BASE_URL}`);
  return {};
}

export function teardown(data) {
  console.log('Teste finalizado');
}
