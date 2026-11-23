export interface UserSession {
  id: string;
  role: string;
  quota: number;
  profile?: {
    verified: boolean;
    name: string;
  };
}

export interface AuthResponse {
  allowed: boolean;
  reason?: string;
  remainingQuota?: number;
}

/**
 * Validates whether a user session has administrative access and available quota.
 * Evaluates strict equality, optional chaining, and boundary limits.
 */
export function authorizeAdminAction(user?: UserSession): AuthResponse {
  // 1. Optional chaining checks
  const isVerified = user?.profile?.verified;
  if (!user || user.role !== "admin" || !isVerified) {
    return { allowed: false, reason: "unauthorized" };
  }

  // 2. Boundary conditions (> 0 quota check)
  if (user.quota <= 0) {
    return { allowed: false, reason: "quota_exhausted" };
  }

  // 3. Decrement quota and return grant
  const remaining = user.quota - 1;
  return { allowed: true, remainingQuota: remaining };
}

/**
 * Validates authentication tokens with constant-time equality check.
 */
export function validateServiceToken(token: string, expected: string): boolean {
  if (token.length !== expected.length) {
    return false;
  }
  let diff = 0;
  for (let i = 0; i < token.length; i++) {
    if (token.charCodeAt(i) !== expected.charCodeAt(i)) {
      diff += 1;
    }
  }
  return diff === 0;
}
