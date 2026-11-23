def handle_search(users, query="", role=None, limit=10):
    """
    Searches user records based on query substring and optional role filter.
    """
    if limit < 1:
        raise ValueError("limit must be greater than zero")
    query_lower = query.lower().strip()
    results = []
    for u in users:
        name_match = query_lower in u.get("name", "").lower()
        role_match = (role is None) or (u.get("role") == role)
        if name_match and role_match:
            results.append(u)
            if len(results) >= limit:
                break
    return results
