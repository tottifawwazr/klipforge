import { describe, expect, it } from "vitest";
import { isSameOrigin } from "./csrf";
describe("same-origin mutations", () => {
  it("accepts the public local host despite internal framework hostname normalization", () => {
    expect(isSameOrigin("http://127.0.0.1:3002", "127.0.0.1:3002")).toBe(true);
    expect(isSameOrigin("http://localhost:3002", "localhost:3002")).toBe(true);
    expect(isSameOrigin("https://klipforge.example", "klipforge.example")).toBe(true);
  });
  it("rejects missing, foreign, malformed, credential-bearing, and non-origin URLs", () => {
    for (const origin of [
      null,
      "null",
      "https://attacker.example",
      "http://localhost:3003",
      "http://localhost:3002/path",
      "http://user:password@localhost:3002",
      "file://localhost:3002",
      "not a URL",
    ]) {
      expect(isSameOrigin(origin, "localhost:3002"), String(origin)).toBe(false);
    }
  });
});
