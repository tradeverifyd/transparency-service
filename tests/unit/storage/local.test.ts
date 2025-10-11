/**
 * Local Filesystem Storage Tests
 * Test suite for local filesystem storage implementation
 */

import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { LocalStorage } from "../../../src/lib/storage/local.ts";
import { StorageNotFoundError } from "../../../src/lib/storage/interface.ts";
import * as fs from "fs";
import * as path from "path";

describe("LocalStorage", () => {
  const testDir = "./.test-storage";
  let storage: LocalStorage;

  beforeEach(async () => {
    // Clean up test directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });
    storage = new LocalStorage(testDir);
  });

  afterEach(() => {
    // Clean up after tests
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
  });

  test("type is 'local'", () => {
    expect(storage.type).toBe("local");
  });

  test("put stores data at key", async () => {
    const key = "test/object";
    const data = new TextEncoder().encode("Hello, world!");

    await storage.put(key, data);

    const filePath = path.join(testDir, key);
    expect(fs.existsSync(filePath)).toBe(true);

    const stored = fs.readFileSync(filePath);
    expect(stored).toEqual(Buffer.from(data));
  });

  test("put creates parent directories", async () => {
    const key = "deeply/nested/path/to/object";
    const data = new TextEncoder().encode("test");

    await storage.put(key, data);

    const filePath = path.join(testDir, key);
    expect(fs.existsSync(filePath)).toBe(true);
  });

  test("get retrieves stored data", async () => {
    const key = "test/object";
    const data = new TextEncoder().encode("Hello, world!");

    await storage.put(key, data);
    const retrieved = await storage.get(key);

    expect(retrieved).toEqual(data);
  });

  test("get returns null for non-existent key", async () => {
    const retrieved = await storage.get("nonexistent/key");
    expect(retrieved).toBeNull();
  });

  test("exists returns true for existing key", async () => {
    const key = "test/object";
    const data = new TextEncoder().encode("test");

    await storage.put(key, data);
    const doesExist = await storage.exists(key);

    expect(doesExist).toBe(true);
  });

  test("exists returns false for non-existent key", async () => {
    const doesExist = await storage.exists("nonexistent/key");
    expect(doesExist).toBe(false);
  });

  test("list returns keys with prefix", async () => {
    // Store multiple objects
    await storage.put("prefix/a", new Uint8Array([1]));
    await storage.put("prefix/b", new Uint8Array([2]));
    await storage.put("prefix/nested/c", new Uint8Array([3]));
    await storage.put("other/d", new Uint8Array([4]));

    const keys = await storage.list("prefix/");

    expect(keys).toContain("prefix/a");
    expect(keys).toContain("prefix/b");
    expect(keys).toContain("prefix/nested/c");
    expect(keys).not.toContain("other/d");
  });

  test("list returns empty array for non-matching prefix", async () => {
    await storage.put("test/object", new Uint8Array([1]));

    const keys = await storage.list("nonexistent/");

    expect(keys).toEqual([]);
  });

  test("delete removes object at key", async () => {
    const key = "test/object";
    await storage.put(key, new Uint8Array([1]));

    expect(await storage.exists(key)).toBe(true);

    await storage.delete(key);

    expect(await storage.exists(key)).toBe(false);
  });

  test("delete does not throw for non-existent key", async () => {
    // Should not throw
    await storage.delete("nonexistent/key");
  });

  test("handles binary data correctly", async () => {
    const key = "binary/data";
    const data = new Uint8Array([0, 1, 2, 255, 254, 253]);

    await storage.put(key, data);
    const retrieved = await storage.get(key);

    expect(retrieved).toEqual(data);
  });

  test("handles large files", async () => {
    const key = "large/file";
    const size = 10 * 1024 * 1024; // 10MB
    const data = new Uint8Array(size);
    // Fill with pattern
    for (let i = 0; i < size; i++) {
      data[i] = i % 256;
    }

    await storage.put(key, data);
    const retrieved = await storage.get(key);

    expect(retrieved?.length).toBe(size);
    expect(retrieved).toEqual(data);
  });
});
