#!/usr/bin/env node
// pool-sheet-append.mjs — append one pool run/model performance row to the Google
// Sheet configured at ~/.config/clorchestrate/sheets.json. Called by the pool-coordinate skill
// once per model label after a run's plan+implementation stages are done.
//
// No npm dependencies: service-account auth is a hand-signed RS256 JWT (node:crypto)
// exchanged for a bearer token, then a plain fetch (built into Node >=18) to the
// Sheets API. Reads one JSON row object from stdin, or as a single JSON-string CLI arg.
//
// Usage:
//   echo '{"date":"2026-09-20", ...}' | pool-sheet-append.mjs
//   pool-sheet-append.mjs '{"date":"2026-09-20", ...}'

import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { createSign } from 'node:crypto';

const CONFIG_PATH = join(homedir(), '.config', 'clorchestrate', 'sheets.json');

// Column order the sheet expects — one row per model per run.
const ROW_FIELDS = [
  'date',
  'run',
  'model',
  'planFeatures',
  'planFeaturesPct',
  'planDurationSeconds',
  'implFeatures',
  'implFeaturesPct',
  'implDurationSeconds',
  'correct',
  'selected',
];

export function validateRow(row) {
  for (const field of ROW_FIELDS) {
    if (row[field] === undefined || row[field] === null) {
      throw new Error(`row is missing required field "${field}"`);
    }
  }
}

export function buildRowArray(row) {
  validateRow(row);
  return ROW_FIELDS.map((field) => row[field]);
}

function base64url(input) {
  return Buffer.from(input).toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

// Hand-rolled service-account JWT (RS256) — avoids pulling in google-auth-library for
// a single signing step.
export function signJwt(serviceAccount, { now = Math.floor(Date.now() / 1000) } = {}) {
  const header = { alg: 'RS256', typ: 'JWT' };
  const claims = {
    iss: serviceAccount.client_email,
    scope: 'https://www.googleapis.com/auth/spreadsheets',
    aud: 'https://oauth2.googleapis.com/token',
    iat: now,
    exp: now + 3600,
  };
  const signingInput = `${base64url(JSON.stringify(header))}.${base64url(JSON.stringify(claims))}`;
  const signature = createSign('RSA-SHA256').update(signingInput).sign(serviceAccount.private_key);
  return `${signingInput}.${base64url(signature)}`;
}

export async function getAccessToken(serviceAccount, { fetchImpl = fetch } = {}) {
  const jwt = signJwt(serviceAccount);
  const res = await fetchImpl('https://oauth2.googleapis.com/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'urn:ietf:params:oauth:grant-type:jwt-bearer',
      assertion: jwt,
    }),
  });
  if (!res.ok) {
    throw new Error(`token exchange failed: ${res.status} ${await res.text()}`);
  }
  const { access_token: accessToken } = await res.json();
  return accessToken;
}

export async function appendRow(row, config, serviceAccount, { fetchImpl = fetch } = {}) {
  const values = buildRowArray(row);
  const accessToken = await getAccessToken(serviceAccount, { fetchImpl });
  const range = encodeURIComponent(`${config.sheetName}!A1`);
  const url = `https://sheets.googleapis.com/v4/spreadsheets/${config.spreadsheetId}/values/${range}:append?valueInputOption=USER_ENTERED`;
  const res = await fetchImpl(url, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${accessToken}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ values: [values] }),
  });
  if (!res.ok) {
    throw new Error(`sheet append failed: ${res.status} ${await res.text()}`);
  }
  return res.json();
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

async function main() {
  const arg = process.argv[2];
  const raw = arg ?? (await readStdin());
  if (!raw || !raw.trim()) {
    throw new Error('no row JSON provided (pass as an arg or pipe via stdin)');
  }
  const row = JSON.parse(raw);

  const config = JSON.parse(readFileSync(CONFIG_PATH, 'utf8'));
  const serviceAccount = JSON.parse(readFileSync(config.serviceAccountKeyFile, 'utf8'));

  await appendRow(row, config, serviceAccount);
  console.log('pool-sheet-append: row appended');
}

const isMain = import.meta.url === `file://${process.argv[1]}`;
if (isMain) {
  main().catch((err) => {
    console.error(`pool-sheet-append: ${err.message}`);
    process.exit(1);
  });
}
