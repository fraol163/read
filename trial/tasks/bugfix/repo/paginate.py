def paginate(items, page, page_size):
    """
    Paginates a list of items using 1-based page numbering.
    """
    if page < 1 or page_size < 1:
        raise ValueError("page and page_size must be positive integers")
    # BUG: starting offset is calculated from page * page_size instead of (page - 1) * page_size
    start = page * page_size
    end = start + page_size
    return items[start:end]
