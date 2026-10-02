import { test } from "node:test";
import assert from "node:assert/strict";
import { greeting } from "./greeting.js";

test("greeting", () => {
  assert.equal(greeting(), "Olá, mundo!");
  assert.equal(greeting("Ana"), "Olá, Ana!");
});
