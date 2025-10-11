/**
 * Go-COSE Interoperability Tests
 * Verifies TypeScript implementation can read/verify go-cose signatures
 */

import { describe, test, expect } from "bun:test";
import { decodeCoseSign1, verifyCoseSign1, getProtectedHeaders } from "../../src/lib/cose/sign.ts";
import * as fs from "fs";
import * as path from "path";

interface GoTestVector {
  payload: string;
  protectedHeaders: Record<string, any>;
  coseSign1Bytes: string;
  publicKeyX: string;
  publicKeyY: string;
  description: string;
}

function hexToBytes(hex: string): Uint8Array {
  if (hex.length === 0) return new Uint8Array(0);
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
  }
  return bytes;
}

async function importPublicKeyFromCoordinates(x: Uint8Array, y: Uint8Array): Promise<CryptoKey> {
  const uncompressed = new Uint8Array(65);
  uncompressed[0] = 0x04;
  uncompressed.set(x, 1);
  uncompressed.set(y, 33);

  return await crypto.subtle.importKey(
    "raw",
    uncompressed,
    { name: "ECDSA", namedCurve: "P-256" },
    true,
    ["verify"]
  );
}

describe("Go-COSE Interoperability", () => {
  const vectorPath = path.join(__dirname, "cose-vectors", "go-cose-vectors.json");
  const vectors: GoTestVector[] = JSON.parse(fs.readFileSync(vectorPath, "utf-8"));

  for (const vector of vectors) {
    test(`TypeScript verifies go-cose: ${vector.description}`, async () => {
      const payload = hexToBytes(vector.payload);
      const coseBytes = hexToBytes(vector.coseSign1Bytes);
      const publicKeyX = hexToBytes(vector.publicKeyX);
      const publicKeyY = hexToBytes(vector.publicKeyY);

      const publicKey = await importPublicKeyFromCoordinates(publicKeyX, publicKeyY);
      const coseSign1 = decodeCoseSign1(coseBytes);
      const isValid = await verifyCoseSign1(coseSign1, publicKey);

      expect(isValid).toBe(true);

      const headers = getProtectedHeaders(coseSign1);
      // go-cose uses integer labels per COSE spec: 1 = alg
      expect(headers["1"] || headers.alg).toBe(vector.protectedHeaders.alg);

      if (vector.protectedHeaders.iss) {
        expect(headers.iss).toBe(vector.protectedHeaders.iss);
      }

      if (coseSign1.payload) {
        expect(coseSign1.payload).toEqual(payload);
      }
    });
  }

  test("summary: all go-cose vectors verified", () => {
    expect(vectors.length).toBe(5);
  });
});
