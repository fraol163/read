use super::Mutant;

/// Detects if a file path is a C++ source or header file.
pub fn is_cpp_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with(".cpp")
        || lower.ends_with(".cc")
        || lower.ends_with(".cxx")
        || lower.ends_with(".h")
        || lower.ends_with(".hpp")
}

/// Generates targeted AST mutation candidates for a single diff line in C++ files.
/// Strictly implements:
/// 1. INVERT_EQUALITY (== <-> !=, < <-> <=, > <-> >=)
/// 2. RETURN_DEFAULT (return x <-> return {}; / return nullptr; / return 0;)
/// 3. BOUNDARY_SHIFT (n <-> n+1/n-1, ++ <-> --)
/// 4. REMOVE_NULLPTR_CHECK (if (ptr != nullptr) <-> if (true))
pub fn generate_cpp_line_mutants(
    line: &str,
    line_num: usize,
    file_path: Option<&str>,
    current_count: usize,
) -> Vec<Mutant> {
    let mut candidates = Vec::new();
    let mut next_id = current_count + 1;

    let trimmed = line.trim();
    if trimmed.is_empty()
        || trimmed.starts_with("//")
        || trimmed.starts_with("/*")
        || trimmed.starts_with('*')
        || trimmed.starts_with("#include")
    {
        return candidates;
    }

    // 4. REMOVE_NULLPTR_CHECK: if (ptr != nullptr) -> if (true)
    for null_token in &["!= nullptr", "!= NULL"] {
        if let Some(null_idx) = line.find(null_token) {
            let before = line[..null_idx].trim_end();
            let mut idx = before.len();
            let bytes = before.as_bytes();
            let mut paren_depth = 0;
            while idx > 0 {
                let prev = bytes[idx - 1];
                if prev == b')' {
                    paren_depth += 1;
                    idx -= 1;
                } else if prev == b'(' {
                    if paren_depth > 0 {
                        paren_depth -= 1;
                        idx -= 1;
                    } else {
                        break;
                    }
                } else if paren_depth == 0 && (prev.is_ascii_whitespace() || prev == b'&' || prev == b'|') {
                    break;
                } else {
                    idx -= 1;
                }
            }

            let var_start = idx;
            let expr_to_replace = format!("{}{}", &line[var_start..null_idx], null_token);
            let mutated = line.replacen(&expr_to_replace, "true", 1);

            candidates.push(Mutant {
                id: next_id,
                original_line: line.to_string(),
                mutated_line: mutated,
                line_number: line_num,
                operator: "REMOVE_NULLPTR_CHECK".to_string(),
                file_path: file_path.map(|s| s.to_string()),
            });
            next_id += 1;
            break;
        }
    }

    // 1. INVERT_EQUALITY: == <-> !=, < <-> <=, > <-> >=
    if line.contains(" == ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" == ", " != ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" != ") && !line.contains("!= nullptr") && !line.contains("!= NULL") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" != ", " == ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    if line.contains(" <= ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" <= ", " > ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" >= ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" >= ", " < ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" < ") && !line.contains("template") && !line.contains("#include") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" < ", " <= ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" > ") && !line.contains("->") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" > ", " >= ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 2. RETURN_DEFAULT: return x <-> return {}; / return nullptr; / return 0;
    if (trimmed.starts_with("return ") || trimmed == "return")
        && !trimmed.contains("return {};")
        && !trimmed.contains("return nullptr;")
        && !trimmed.contains("return NULL;")
        && !trimmed.contains("return 0;")
    {
        let indent: String = line.chars().take_while(|c| c.is_whitespace()).collect();
        let replacement = if trimmed.contains("ptr") || trimmed.contains('*') {
            "return nullptr;"
        } else {
            "return {};"
        };
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: format!("{}{}", indent, replacement),
            line_number: line_num,
            operator: "RETURN_DEFAULT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 3. BOUNDARY_SHIFT: n <-> n+1/n-1, ++ <-> --
    if line.contains(" + 1") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" + 1", " - 1", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" - 1") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" - 1", " + 1", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    if line.contains("++") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("++", "--", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    } else if line.contains("--") && !line.starts_with("---") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("--", "++", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    }

    candidates
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_remove_nullptr_check() {
        let m = generate_cpp_line_mutants("if (buffer != nullptr) {", 10, Some("Buffer.cpp"), 0);
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].operator, "REMOVE_NULLPTR_CHECK");
        assert_eq!(m[0].mutated_line, "if (true) {");
    }

    #[test]
    fn test_invert_equality() {
        let m1 = generate_cpp_line_mutants("if (status == 200) {", 12, Some("Http.cpp"), 0);
        assert_eq!(m1.len(), 1);
        assert_eq!(m1[0].operator, "INVERT_EQUALITY");
        assert_eq!(m1[0].mutated_line, "if (status != 200) {");

        let m2 = generate_cpp_line_mutants("if (count <= 10) {", 14, Some("Http.cpp"), 0);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "INVERT_EQUALITY");
        assert_eq!(m2[0].mutated_line, "if (count > 10) {");

        let m3 = generate_cpp_line_mutants("if (offset < limit) {", 16, Some("Http.cpp"), 0);
        assert_eq!(m3.len(), 1);
        assert_eq!(m3[0].operator, "INVERT_EQUALITY");
        assert_eq!(m3[0].mutated_line, "if (offset <= limit) {");
    }

    #[test]
    fn test_return_default() {
        let m = generate_cpp_line_mutants("    return result;", 20, Some("Service.cpp"), 0);
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].operator, "RETURN_DEFAULT");
        assert_eq!(m[0].mutated_line, "    return {};");

        let m2 = generate_cpp_line_mutants("    return raw_ptr;", 22, Some("Service.cpp"), 0);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "RETURN_DEFAULT");
        assert_eq!(m2[0].mutated_line, "    return nullptr;");
    }

    #[test]
    fn test_boundary_shift() {
        let m = generate_cpp_line_mutants("i++;", 30, Some("Loop.cpp"), 0);
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].operator, "BOUNDARY_SHIFT");
        assert_eq!(m[0].mutated_line, "i--;");

        let m2 = generate_cpp_line_mutants("int next = current + 1;", 32, Some("Loop.cpp"), 0);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "BOUNDARY_SHIFT");
        assert_eq!(m2[0].mutated_line, "int next = current - 1;");
    }
}
