package com.example.service;

public class OrderValidationService {

    public static class Order {
        private final String id;
        private final String role;
        private final int quantity;

        public Order(String id, String role, int quantity) {
            this.id = id;
            this.role = role;
            this.quantity = quantity;
        }

        public String getId() { return id; }
        public String getRole() { return role; }
        public int getQuantity() { return quantity; }
    }

    public static class ValidationResult {
        private final boolean valid;
        private final int allocatedQuantity;

        public ValidationResult(boolean valid, int allocatedQuantity) {
            this.valid = valid;
            this.allocatedQuantity = allocatedQuantity;
        }

        public boolean isValid() { return valid; }
        public int getAllocatedQuantity() { return allocatedQuantity; }
    }

    public ValidationResult validateAndAllocate(Order order) {
        // 1. Null check operator
        if (order != null) {
            // 2. Strict equality operator
            if (order.getRole().equals("ADMIN")) {
                // 3. Boundary shift operator
                if (order.getQuantity() > 0) {
                    int remaining = order.getQuantity() - 1;
                    // 4. Return null operator
                    return new ValidationResult(true, remaining);
                }
            }
        }
        return new ValidationResult(false, 0);
    }
}
