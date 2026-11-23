package com.example.service;

public class OrderValidationServiceTest {

    public static void main(String[] args) {
        OrderValidationService service = new OrderValidationService();

        // 1. Test null order -> must be invalid
        OrderValidationService.ValidationResult resNull = service.validateAndAllocate(null);
        if (resNull == null || resNull.isValid()) {
            throw new AssertionError("Null order must fail validation");
        }

        // 2. Test non-admin role -> must be invalid
        OrderValidationService.Order userOrder = new OrderValidationService.Order("ord_1", "USER", 5);
        OrderValidationService.ValidationResult resUser = service.validateAndAllocate(userOrder);
        if (resUser == null || resUser.isValid()) {
            throw new AssertionError("Non-admin role must fail validation");
        }

        // 3. Test admin with zero quantity -> must fail boundary
        OrderValidationService.Order zeroOrder = new OrderValidationService.Order("ord_2", "ADMIN", 0);
        OrderValidationService.ValidationResult resZero = service.validateAndAllocate(zeroOrder);
        if (resZero == null || resZero.isValid()) {
            throw new AssertionError("Zero quantity must fail boundary check");
        }

        // 4. Test admin with negative quantity -> must fail boundary
        OrderValidationService.Order negOrder = new OrderValidationService.Order("ord_3", "ADMIN", -1);
        OrderValidationService.ValidationResult resNeg = service.validateAndAllocate(negOrder);
        if (resNeg == null || resNeg.isValid()) {
            throw new AssertionError("Negative quantity must fail boundary check");
        }

        // 5. Test valid admin with quantity 5 -> must pass and remaining == 4
        OrderValidationService.Order validOrder = new OrderValidationService.Order("ord_4", "ADMIN", 5);
        OrderValidationService.ValidationResult resValid = service.validateAndAllocate(validOrder);
        if (resValid == null || !resValid.isValid()) {
            throw new AssertionError("Valid admin order must pass validation");
        }
        if (resValid.getAllocatedQuantity() != 4) {
            throw new AssertionError("Allocated quantity must decrement by 1 (expected 4, got " + resValid.getAllocatedQuantity() + ")");
        }

        System.out.println("All 5 OrderValidationServiceTest assertions passed cleanly.");
    }
}
