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

    group('Accepts types case-insensitively and preserves their casing', () => {
      // The fake upstream replies with this marker body when the user prompt
      // contains "Generate a HTML file" with the original uppercase casing.
      const res = postRequest({ type: 'HTML', spec: 'a greeting' });

      check(res, {
        'status is 200': (r) => r.status === 200,
        'content type is correct': (r) => r.headers['Content-Type'] === 'text/html; charset=utf-8',
        'content disposition is correct': (r) => r.headers['Content-Disposition'] === 'attachment; filename="representation.html"',
        'generator received the original casing': (r) => r.body === 'uppercase-type-preserved',
      });
    });
  });

  group('Request Validation', () => {
    const requiredFieldsError = "fields 'type' and 'spec' are required";

    group('Rejects requests with missing fields', () => {
      const res = postRequest({ type: 'html' });

      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns the exact error message': (r) => getJsonError(r.body) === requiredFieldsError,
      });
    });

    group('Rejects requests with empty spec (after whitespace trimming)', () => {
      const res = postRequest({ type: 'html', spec: '   ' });

      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns the exact error message': (r) => getJsonError(r.body) === requiredFieldsError,
      });
    });

    group('Rejects a null JSON body as missing fields', () => {
      const res = postRequest('null');

      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns the exact error message': (r) => getJsonError(r.body) === requiredFieldsError,
      });
    });

    group('Rejects unsupported types', () => {
      const res = postRequest({ type: 'json', spec: 'a config file' });

      check(res, {
        'status is 422': (r) => r.status === 422,
        'returns the exact error message': (r) =>
          getJsonError(r.body) === 'unsupported type "json": only "html" and "markdown" are supported',
      });
    });

    group('Rejects bodies larger than 1 MiB', () => {
      const res = postRequest({ type: 'html', spec: 'a'.repeat(1024 * 1024) });

      check(res, {
        'status is 400': (r) => r.status === 400,
        'error begins with the decoding prefix': (r) => getJsonError(r.body).startsWith('invalid request body: '),
      });
    });

    group('Rejects malformed JSON bodies', () => {
      const res = postRequest('{not json');

      check(res, {
        'status is 400': (r) => r.status === 400,
        'error begins with the decoding prefix': (r) => getJsonError(r.body).startsWith('invalid request body: '),
      });
    });

    group('Rejects trailing content after the JSON object', () => {
      const res = postRequest('{"type":"html","spec":"x"} extra');

      check(res, {
        'status is 400': (r) => r.status === 400,
        'error begins with the decoding prefix': (r) => getJsonError(r.body).startsWith('invalid request body: '),
      });
    });

    group('Rejects non-POST HTTP methods', () => {
      const res = http.get(ENDPOINT);

      check(res, {
        'status is 405': (r) => r.status === 405,
        'allows POST': (r) => r.headers['Allow'] === 'POST',
        'body is plain text': (r) => r.headers['Content-Type'] === 'text/plain; charset=utf-8',
      });
    });
  });

  group('Routing', () => {
    group('Redirects the bare /v1 path', () => {
      const res = http.get(`${API_URL}/v1`, { redirects: 0 });

      check(res, {
        'status is 307': (r) => r.status === 307,
        'redirects to /v1/': (r) => r.headers['Location'] === '/v1/',
      });
    });

    group('Rejects unknown paths', () => {
      const res = http.get(`${API_URL}/unknown`);

      check(res, {
        'status is 404': (r) => r.status === 404,
        'body is plain text': (r) => r.headers['Content-Type'] === 'text/plain; charset=utf-8',
      });
    });
  });

  group('Error Handling', () => {
    group('Surfaces upstream generation failures as 500s', () => {
      const res = postRequest({ type: 'html', spec: 'fail_generation' });

      check(res, {
        'status is 500': (r) => r.status === 500,
        'error begins with the generation prefix': (r) => getJsonError(r.body).startsWith('generation failed: '),
      });
    });
  });
}

