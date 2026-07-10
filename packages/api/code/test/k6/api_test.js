import http from 'k6/http';
import { check, group } from 'k6';

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const ENDPOINT = `${API_URL}/v1/representation`;

// Fail the k6 run (non-zero exit) when any check fails, so the Go test
// wrapper reports the failure; bare checks do not affect k6's exit code.
export const options = {
  thresholds: {
    checks: ['rate==1'],
  },
};

// Helper for sending POST requests with JSON headers
function postRequest(payload) {
  const body = typeof payload === 'string' ? payload : JSON.stringify(payload);
  return http.post(ENDPOINT, body, {
    headers: { 'Content-Type': 'application/json' },
  });
}

// Helper to safely extract error message from JSON response
function getJsonError(resBody) {
  try {
    return JSON.parse(resBody).error || '';
  } catch (e) {
    return '';
  }
}

export default function () {
  group('Happy Path', () => {
    group('Generates HTML successfully', () => {
      const res = postRequest({ type: 'html', spec: 'a greeting' });
      
      check(res, {
        'status is 200': (r) => r.status === 200,
        'content type is correct': (r) => r.headers['Content-Type'] === 'text/html; charset=utf-8',
        'content disposition is correct': (r) => r.headers['Content-Disposition'] === 'attachment; filename="representation.html"',
        'body is not empty': (r) => r.body && r.body.length > 0,
      });
    });

    group('Generates Markdown successfully', () => {
      const res = postRequest({ type: 'markdown', spec: 'a greeting' });
      
      check(res, {
        'status is 200': (r) => r.status === 200,
        'content type is correct': (r) => r.headers['Content-Type'] === 'text/markdown; charset=utf-8',
        'content disposition is correct': (r) => r.headers['Content-Disposition'] === 'attachment; filename="representation.md"',
        'body is not empty': (r) => r.body && r.body.length > 0,
      });
    });
  });

  group('Request Validation', () => {
    group('Rejects requests with missing fields', () => {
      const res = postRequest({ type: 'html' });
      
      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });

    group('Rejects requests with empty spec (after whitespace trimming)', () => {
      const res = postRequest({ type: 'html', spec: '   ' });
      
      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });

    group('Rejects unsupported types', () => {
      const res = postRequest({ type: 'json', spec: 'a config file' });
      
      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });

    group('Rejects bodies larger than 1 MiB', () => {
      const res = postRequest({ type: 'html', spec: 'a'.repeat(1024 * 1024) });

      check(res, {
        'status is 400': (r) => r.status === 400,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });

    group('Rejects malformed JSON bodies', () => {
      const res = postRequest('{not json');
      
      check(res, {
        'status is 400': (r) => r.status === 400,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });

    group('Rejects non-POST HTTP methods', () => {
      const res = http.get(ENDPOINT);
      
      check(res, {
        'status is 405': (r) => r.status === 405,
      });
    });
  });

  group('Error Handling', () => {
    group('Surfaces upstream generation failures as 500s', () => {
      const res = postRequest({ type: 'html', spec: 'fail_generation' });
      
      check(res, {
        'status is 500': (r) => r.status === 500,
        'returns valid error message': (r) => getJsonError(r.body) !== '',
      });
    });
  });
}

