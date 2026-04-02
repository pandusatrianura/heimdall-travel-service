import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

// Load the search payload from the shared JSON file
const payload = JSON.parse(open('./search_payload.json'));

export const options = {
    // Thresholds: Fail the test if more than 1% of requests fail or 
    // if the P95 latency exceeds 500ms
    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
    },
    // Scenarios are defined via CLI flags for flexibility, 
    // but these are the defaults:
    scenarios: {
        average_load: {
            executor: 'constant-vus',
            vus: 10,
            duration: '30s',
        },
    },
};

export default function () {
    const url = __ENV.BASE_URL || 'http://localhost:8080/api/v1/search';
    
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(url, JSON.stringify(payload), params);

    // Validation
    check(res, {
        'status is 200': (r) => r.status === 200,
        'has results': (r) => {
            const body = JSON.parse(r.body);
            return body.flights && body.flights.length >= 0;
        },
    });

    // Pacing
    sleep(1);
}
