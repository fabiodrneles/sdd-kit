import { test } from "node:test";
import assert from "node:assert/strict";

const sum = (a, b) => a + b;

test("sum", () => {
  assert.equal(sum(2, 3), 5);
});
