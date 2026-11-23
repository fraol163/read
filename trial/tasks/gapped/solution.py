def apply_rebate(amount, tier):
    if amount < 0:
        raise ValueError("negative amount")
    bonus = 0
    if tier == "gold":
        bonus = 20
    elif tier == "silver":
        bonus = 10
    return amount + bonus
