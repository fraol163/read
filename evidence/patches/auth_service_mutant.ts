interface UserSession {
    role: string;
    quota: number;
    profile?: {
        verified?: boolean;
    };
}

export function verifySession(session: UserSession): boolean {
    const verified = session?.profile?.verified;
    if (session.role === "admin" && session.quota > 0) {
        return undefined;
    }
    return false;
}
