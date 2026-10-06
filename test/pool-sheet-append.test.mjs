// Unit test for bin/pool-sheet-append.mjs. No real Google API calls: fetch is stubbed
// via node:test's built-in mock.method. Covers the pure row-building/validation and the
// happy-path append call (token exchange + sheet append) with a stubbed fetch.

import { test, describe, mock } from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPairSync } from 'node:crypto';
import {
  buildRowArray,
  validateRow,
  signJwt,
  appendRow,
} from '../bin/pool-sheet-append.mjs';

const ROW = {
  date: '2026-09-20',
  run: 'run-abc',
  model: 'claude',
  planFeatures: 'auth, logging',
  planFeaturesPct: 66.7,
  planDurationSeconds: 120,
  implFeatures: 'auth',
  implFeaturesPct: 50,
  implDurationSeconds: 300,
  correct: true,
  selected: true,
};

describe('pool-sheet-append.mjs', () => {
  test('buildRowArray puts fields in the fixed column order', () => {
    assert.deepEqual(buildRowArray(ROW), [
      '2026-09-20',
      'run-abc',
      'claude',
      'auth, logging',
      66.7,
      120,
      'auth',
      50,
      300,
      true,
      true,
    ]);
  });

  test('validateRow throws on a missing field', () => {
    const { implDurationSeconds, ...incomplete } = ROW;
    assert.throws(() => validateRow(incomplete), /implDurationSeconds/);
  });

  test('appendRow exchanges a token then posts the row, using the stubbed fetch', async () => {
    const { privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
    const serviceAccount = {
      client_email: 'test@example.iam.gserviceaccount.com',
      private_key: privateKey.export({ type: 'pkcs1', format: 'pem' }),
    };
    const config = { spreadsheetId: 'sheet123', sheetName: 'Pool Runs' };

    const calls = [];
    const fetchImpl = mock.fn(async (url, opts) => {
      calls.push({ url, opts });
      if (url === 'https://oauth2.googleapis.com/token') {
        return { ok: true, json: async () => ({ access_token: 'fake-token' }) };
      }
      return { ok: true, json: async () => ({ updates: { updatedRows: 1 } }) };
    });

    const result = await appendRow(ROW, config, serviceAccount, { fetchImpl });

    assert.equal(calls.length, 2);
    assert.equal(calls[0].url, 'https://oauth2.googleapis.com/token');
    assert.match(calls[1].url, /spreadsheets\/sheet123\/values\/Pool%20Runs!A1:append/);
    assert.equal(calls[1].opts.headers.Authorization, 'Bearer fake-token');
    assert.deepEqual(JSON.parse(calls[1].opts.body).values[0], buildRowArray(ROW));
    assert.deepEqual(result, { updates: { updatedRows: 1 } });
  });

  test('signJwt produces a three-part JWT string', () => {
    const { privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
    const jwt = signJwt({
      client_email: 'test@example.iam.gserviceaccount.com',
      private_key: privateKey.export({ type: 'pkcs1', format: 'pem' }),
    });
    assert.equal(jwt.split('.').length, 3);
  });
});
