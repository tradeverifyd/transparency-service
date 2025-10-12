/**
 * Issuer Resolver
 * Single choke point for issuer key discovery
 * Fetches public keys from issuer URLs with caching
 */

import type { JWK } from "./key-material.ts";
import { importPublicKeyFromJWK } from "./key-material.ts";

/**
 * Cache entry
 */
interface CacheEntry {
  keys: JWK[];
  timestamp: number;
}

/**
 * Resolver options
 */
interface ResolverOptions {
  ttl?: number; // Time-to-live in milliseconds (default: 3600000 = 1 hour)
  wellKnownPath?: string; // Custom .well-known path
}

/**
 * In-memory cache for issuer keys
 */
const issuerCache = new Map<string, CacheEntry>();

/**
 * Default TTL: 1 hour
 */
const DEFAULT_TTL = 3600000;

/**
 * Resolve issuer public key from URL
 * Single choke point for all key discovery
 *
 * @param issuerUrl - Issuer URL (e.g., "https://example.com")
 * @param wellKnownPath - Custom .well-known path (default: "/.well-known/jwks.json")
 * @param options - Resolver options
 * @returns Public CryptoKey (first key in set)
 */
export async function resolveIssuer(
  issuerUrl: string,
  wellKnownPath: string = "/.well-known/jwks.json",
  options: ResolverOptions = {}
): Promise<CryptoKey> {
  const jwks = await fetchIssuerKeys(issuerUrl, wellKnownPath, options);

  if (jwks.length === 0) {
    throw new Error(`No keys found for issuer: ${issuerUrl}`);
  }

  // Return first key
  const jwk = jwks[0]!;
  return await importPublicKeyFromJWK(jwk);
}

/**
 * Resolve specific key by kid
 *
 * @param issuerUrl - Issuer URL
 * @param kid - Key ID
 * @param wellKnownPath - Custom .well-known path
 * @param options - Resolver options
 * @returns Public CryptoKey matching kid
 */
export async function resolveKeyByKid(
  issuerUrl: string,
  kid: string,
  wellKnownPath: string = "/.well-known/jwks.json",
  options: ResolverOptions = {}
): Promise<CryptoKey> {
  const jwks = await fetchIssuerKeys(issuerUrl, wellKnownPath, options);

  const jwk = jwks.find((k) => k.kid === kid);

  if (!jwk) {
    throw new Error(`Key with kid "${kid}" not found for issuer: ${issuerUrl}`);
  }

  return await importPublicKeyFromJWK(jwk);
}

/**
 * Fetch issuer keys (with caching)
 *
 * @param issuerUrl - Issuer URL
 * @param wellKnownPath - .well-known path
 * @param options - Resolver options
 * @returns Array of JWKs
 */
async function fetchIssuerKeys(
  issuerUrl: string,
  wellKnownPath: string,
  options: ResolverOptions
): Promise<JWK[]> {
  const ttl = options.ttl ?? DEFAULT_TTL;
  const cacheKey = `${issuerUrl}${wellKnownPath}`;

  // Check cache
  const cached = issuerCache.get(cacheKey);
  if (cached && Date.now() - cached.timestamp < ttl) {
    return cached.keys;
  }

  // Fetch from issuer
  const url = new URL(wellKnownPath, issuerUrl).toString();

  let response: Response;
  try {
    response = await fetch(url);
  } catch (error) {
    throw new Error(`Failed to fetch keys from ${url}: ${error}`);
  }

  if (!response.ok) {
    throw new Error(`Failed to fetch keys from ${url}: ${response.status} ${response.statusText}`);
  }

  // Parse JWKS
  let jwks: { keys: unknown[] };
  try {
    jwks = await response.json();
  } catch (error) {
    throw new Error(`Invalid JSON response from ${url}: ${error}`);
  }

  if (!jwks.keys || !Array.isArray(jwks.keys)) {
    throw new Error(`Invalid JWKS format from ${url}: missing "keys" array`);
  }

  // Validate JWKs
  const validKeys: JWK[] = [];
  for (const key of jwks.keys) {
    if (isValidJWK(key)) {
      validKeys.push(key as JWK);
    }
  }

  if (validKeys.length === 0) {
    throw new Error(`No valid JWKs found in response from ${url}`);
  }

  // Cache the keys
  issuerCache.set(cacheKey, {
    keys: validKeys,
    timestamp: Date.now(),
  });

  return validKeys;
}

/**
 * Validate JWK structure
 */
function isValidJWK(key: unknown): key is JWK {
  if (typeof key !== "object" || key === null) {
    return false;
  }

  const jwk = key as Record<string, unknown>;

  // Required fields for EC keys
  if (jwk.kty !== "EC") {
    return false;
  }

  if (typeof jwk.crv !== "string") {
    return false;
  }

  if (typeof jwk.x !== "string") {
    return false;
  }

  // y is optional for compressed keys, but typically present
  if (jwk.y !== undefined && typeof jwk.y !== "string") {
    return false;
  }

  return true;
}

/**
 * Clear cache
 * Useful for testing or forced refresh
 */
export function clearCache(): void {
  issuerCache.clear();
}

/**
 * Get cache statistics
 */
export function getCacheStats(): { size: number; entries: string[] } {
  return {
    size: issuerCache.size,
    entries: Array.from(issuerCache.keys()),
  };
}
