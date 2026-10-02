import { test } from "node:test";
import assert from "node:assert/strict";
import { sum } from "./sum.js";

test("sum", () => {
  assert.equal(sum(2, 3), 5);
});
