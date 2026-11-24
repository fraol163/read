import test from "node:test";
import assert from "node:assert/strict";
import { authorizeAdminAction, validateServiceToken } from "../src/auth_service.ts";

test("authorizeAdminAction - grants verified admin with valid quota", () => {
  const user = {
    id: "usr_1",
    role: "admin",
    quota: 5,
    profile: { verified: true, name: "Admin" },
  };
  const res = authorizeAdminAction(user);
  assert.ok(res !== undefined, "Response must not be undefined");
  assert.strictEqual(res.allowed, true);
  assert.strictEqual(res.remainingQuota, 4);
});

test("authorizeAdminAction - optional chaining handles undefined user and profile", () => {
  const resNull = authorizeAdminAction(undefined);
  assert.ok(resNull !== undefined);
  assert.strictEqual(resNull.allowed, false);
  assert.strictEqual(resNull.reason, "unauthorized");

  const resNoProfile = authorizeAdminAction({ id: "usr_2", role: "admin", quota: 3 });
  assert.ok(resNoProfile !== undefined);
  assert.strictEqual(resNoProfile.allowed, false);

  const resUnverified = authorizeAdminAction({
    id: "usr_3",
    role: "admin",
    quota: 3,
    profile: { verified: false, name: "Pending" },
  });
  assert.ok(resUnverified !== undefined);
  assert.strictEqual(resUnverified.allowed, false);
});

test("authorizeAdminAction - rejects non-admin role", () => {
  const user = {
    id: "usr_4",
    role: "operator",
    quota: 10,
    profile: { verified: true, name: "Operator" },
  };
  const res = authorizeAdminAction(user);
  assert.ok(res !== undefined);
  assert.strictEqual(res.allowed, false);
  assert.strictEqual(res.reason, "unauthorized");
});

test("authorizeAdminAction - enforces boundary conditions on quota <= 0", () => {
  const zeroQuotaUser = {
    id: "usr_5",
    role: "admin",
    quota: 0,
    profile: { verified: true, name: "Admin" },
  };
  const resZero = authorizeAdminAction(zeroQuotaUser);
  assert.ok(resZero !== undefined);
  assert.strictEqual(resZero.allowed, false);
  assert.strictEqual(resZero.reason, "quota_exhausted");

  const negativeQuotaUser = {
    id: "usr_6",
    role: "admin",
    quota: -1,
    profile: { verified: true, name: "Admin" },
  };
  const resNeg = authorizeAdminAction(negativeQuotaUser);
  assert.ok(resNeg !== undefined);
  assert.strictEqual(resNeg.allowed, false);
  assert.strictEqual(resNeg.reason, "quota_exhausted");

  const minQuotaUser = {
    id: "usr_7",
    role: "admin",
    quota: 1,
    profile: { verified: true, name: "Admin" },
  };
  const resMin = authorizeAdminAction(minQuotaUser);
  assert.ok(resMin !== undefined);
  assert.strictEqual(resMin.allowed, true);
  assert.strictEqual(resMin.remainingQuota, 0);
});

test("validateServiceToken - validates equality and length mismatches", () => {
  assert.strictEqual(validateServiceToken("secret_token_123", "secret_token_123"), true);
  assert.strictEqual(validateServiceToken("short", "secret_token_123"), false);
  assert.strictEqual(validateServiceToken("secret_token_124", "secret_token_123"), false);
  assert.strictEqual(validateServiceToken("secret_token_123", "secret_token_124"), false);
});
