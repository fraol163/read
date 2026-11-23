using System;

namespace Example.Service
{
    public class UserPrincipal
    {
        public string? Role { get; set; }
    }

    public class TokenValidationService
    {
        public bool ValidateAdminAccess(UserPrincipal? user)
        {
            var role = user?.Role;
            if (role != null && role.Equals("ADMIN"))
            {
                return true;
            }
            return false;
        }

        public bool ValidateStatus(int status)
        {
            if (status == 200)
            {
                return true;
            }
            return false;
        }

        public bool CheckRetryLimit(int retries)
        {
            if (retries + 1 > 3)
            {
                return false;
            }
            return true;
        }

        public string FormatToken(string rawToken)
        {
            var token = $"TOKEN:{rawToken.Trim()}";
            return token;
        }
    }
}
