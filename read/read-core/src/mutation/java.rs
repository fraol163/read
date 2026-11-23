use super::Mutant;

/// Detects if a file path is a Java source file.
pub fn is_java_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with(".java")
}

/// Generates targeted AST mutation candidates for a single diff line in Java files.
/// Strictly implements:
/// 1. INVERT_EQUALITY (== <-> != and .equals() <-> !.equals())
/// 2. RETURN_NULL (return x <-> return null)
/// 3. BOUNDARY_SHIFT (n <-> n+1/n-1, < <-> <=, > <-> >=)
/// 4. REMOVE_NULL_CHECK (if (x != null) <-> if (true))
pub fn generate_java_line_mutants(
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
    {
        return candidates;
    }

    // 4. REMOVE_NULL_CHECK: if (x != null) -> if (true)
    if let Some(null_idx) = line.find("!= null") {
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
        let expr_to_replace = format!("{}!= null", &line[var_start..null_idx]);
        let mutated = line.replacen(&expr_to_replace, "true", 1);

        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: mutated,
            line_number: line_num,
            operator: "REMOVE_NULL_CHECK".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 1. INVERT_EQUALITY: == <-> != and .equals() <-> !.equals()
    if let Some(eq_idx) = line.find(".equals(") {
        let before = &line[..eq_idx];
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
            } else if paren_depth == 0 && (prev.is_ascii_whitespace() || prev == b'&' || prev == b'|' || prev == b'!') {
                break;
            } else {
                idx -= 1;
            }
        }

        let expr_start = idx;
        let has_bang = expr_start > 0 && bytes[expr_start - 1] == b'!';

        let mutated = if has_bang {
            format!("{}{}", &line[..expr_start - 1], &line[expr_start..])
        } else {
            format!("{}!{}", &line[..expr_start], &line[expr_start..])
        };

        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: mutated,
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

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
    } else if line.contains(" != ") && !line.contains("!= null") {
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

    // 2. RETURN_NULL: return x <-> return null
    if (trimmed.starts_with("return ") || trimmed == "return") && !trimmed.contains("return null") {
        let indent: String = line.chars().take_while(|c| c.is_whitespace()).collect();
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: format!("{}return null;", indent),
            line_number: line_num,
            operator: "RETURN_NULL".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 3. BOUNDARY_SHIFT: n <-> n+1/n-1, < <-> <=, > <-> >=
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
        next_id += 1;
    } else if line.contains("--") && !line.starts_with("---") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("--", "++", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
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
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    } else if line.contains(" >= ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" >= ", " < ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    } else if line.contains(" < ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" < ", " <= ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    } else if line.contains(" > ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" > ", " >= ", 1),
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
    fn test_invert_equality() {
        let m1 = generate_java_line_mutants("if (role.equals(\"ADMIN\")) {", 10, Some("Auth.java"), 0);
        assert_eq!(m1.len(), 1);
        assert_eq!(m1[0].operator, "INVERT_EQUALITY");
        assert_eq!(m1[0].mutated_line, "if (!role.equals(\"ADMIN\")) {");

        let m2 = generate_java_line_mutants("if (!role.equals(\"ADMIN\")) {", 11, Some("Auth.java"), 1);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "INVERT_EQUALITY");
        assert_eq!(m2[0].mutated_line, "if (role.equals(\"ADMIN\")) {");

        let m3 = generate_java_line_mutants("if (status == 200) {", 12, Some("Auth.java"), 2);
        assert_eq!(m3.len(), 1);
        assert_eq!(m3[0].operator, "INVERT_EQUALITY");
        assert_eq!(m3[0].mutated_line, "if (status != 200) {");
    }

    #[test]
    fn test_return_null() {
        let m = generate_java_line_mutants("    return ResponseEntity.ok(user);", 20, Some("Controller.java"), 0);
        let ret_mutant = m.iter().find(|x| x.operator == "RETURN_NULL").unwrap();
        assert_eq!(ret_mutant.mutated_line, "    return null;");
    }

    #[test]
    fn test_boundary_shift() {
        let m1 = generate_java_line_mutants("int next = count + 1;", 30, Some("Counter.java"), 0);
        assert!(m1.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("count - 1")));

        let m2 = generate_java_line_mutants("if (quota < 10) {", 31, Some("Counter.java"), 1);
        assert!(m2.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("quota <= 10")));

        let m3 = generate_java_line_mutants("if (quota > 0) {", 32, Some("Counter.java"), 2);
        assert!(m3.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("quota >= 0")));
    }

    #[test]
    fn test_remove_null_check() {
        let m = generate_java_line_mutants("if (user != null) {", 40, Some("Service.java"), 0);
        let null_mutant = m.iter().find(|x| x.operator == "REMOVE_NULL_CHECK").unwrap();
        assert_eq!(null_mutant.mutated_line, "if (true) {");

        let m2 = generate_java_line_mutants("if (order != null && order.getRole().equals(\"ADMIN\")) {", 41, Some("Service.java"), 1);
        assert!(m2.iter().any(|x| x.operator == "REMOVE_NULL_CHECK" && x.mutated_line.contains("if (true && order.getRole().equals(\"ADMIN\")) {")));
        assert!(m2.iter().any(|x| x.operator == "INVERT_EQUALITY" && x.mutated_line.contains("!order.getRole().equals(\"ADMIN\")")));
    }
}
