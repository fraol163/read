using System;
using Example.Service;
using Xunit;

namespace Example.Service.Tests
{
    public class TokenValidationServiceTest
    {
        private readonly TokenValidationService _service = new TokenValidationService();

        [Fact]
        public void TestNullUserHandling()
        {
            // Verifies null conditional safe access: user?.Role
            bool result = _service.ValidateAdminAccess(null);
            Assert.False(result);
        }

        [Fact]
        public void TestAdminRoleEquality()
        {
            // Verifies role.Equals("ADMIN")
            var adminUser = new UserPrincipal { Role = "ADMIN" };
            var guestUser = new UserPrincipal { Role = "GUEST" };

            Assert.True(_service.ValidateAdminAccess(adminUser));
            Assert.False(_service.ValidateAdminAccess(guestUser));
        }

        [Fact]
        public void TestStatusEquality()
        {
            // Verifies status == 200
            Assert.True(_service.ValidateStatus(200));
            Assert.False(_service.ValidateStatus(401));
            Assert.False(_service.ValidateStatus(500));
        }

        [Fact]
        public void TestRetryLimitBoundary()
        {
            // Verifies retries + 1 > 3 boundary check
            Assert.True(_service.CheckRetryLimit(0));
            Assert.True(_service.CheckRetryLimit(1));
            Assert.True(_service.CheckRetryLimit(2));
            Assert.False(_service.CheckRetryLimit(3));
        }

        [Fact]
        public void TestFormatTokenNonDefault()
        {
            // Verifies return token does not return default/null
            string formatted = _service.FormatToken("abc-123");
            Assert.NotNull(formatted);
            Assert.Equal("TOKEN:abc-123", formatted);
        }
    }
}
