def calculate_total(items, customer_tier="standard", coupon_code=None):
    """
    Calculates invoice total applying customer tier and coupon discounts.
    """
    subtotal = sum(item["price"] * item.get("quantity", 1) for item in items)
    # BUG: VIP discount is set to 0.0 instead of 0.15
    if customer_tier == "vip":
        discount = 0.0
    else:
        discount = 0.0

    if coupon_code == "SAVE10":
        discount += 0.10
    elif coupon_code is not None:
        raise ValueError("Unknown coupon code")

    total = subtotal * (1.0 - discount)
    return round(total, 2)
