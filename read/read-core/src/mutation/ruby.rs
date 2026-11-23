use super::Mutant;

/// Detects if a file path is a Ruby source file.
pub fn is_ruby_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with(".rb")
}

/// Generates targeted AST mutation candidates for a single diff line in Ruby files.
/// Strictly implements:
/// 1. INVERT_EQUALITY (== <-> !=, === <-> !==, eql? <-> !eql?)
/// 2. RETURN_NIL (return x <-> return nil)
/// 3. BOUNDARY_SHIFT (n <-> n+1/n-1, < <-> <=, > <-> >=)
/// 4. REMOVE_SAFE_NAVIGATION (x&.y <-> x.y)
pub fn generate_ruby_line_mutants(
    line: &str,
    line_num: usize,
    file_path: Option<&str>,
    current_count: usize,
) -> Vec<Mutant> {
    let mut candidates = Vec::new();
    let mut next_id = current_count + 1;

    let trimmed = line.trim();
    if trimmed.is_empty()
        || trimmed.starts_with('#')
        || trimmed.starts_with("=begin")
        || trimmed.starts_with("=end")
    {
        return candidates;
    }

    // 4. REMOVE_SAFE_NAVIGATION: x&.y -> x.y
    if line.contains("&.") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("&.", ".", 1),
            line_number: line_num,
            operator: "REMOVE_SAFE_NAVIGATION".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 1. INVERT_EQUALITY: === <-> !==, == <-> !=, eql? <-> !eql?
    if let Some(eql_idx) = line.find(".eql?(") {
        let before = &line[..eql_idx];
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

    if line.contains(" === ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" === ", " !== ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" !== ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" !== ", " === ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" == ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" == ", " != ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" != ") {
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
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" >= ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" >= ", " < ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" < ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" < ", " <= ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" > ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" > ", " >= ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 2. RETURN_NIL: return x <-> return nil
    if (trimmed.starts_with("return ") || trimmed == "return") && !trimmed.contains("return nil") {
        let indent: String = line.chars().take_while(|c| c.is_whitespace()).collect();
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: format!("{}return nil", indent),
            line_number: line_num,
            operator: "RETURN_NIL".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 3. BOUNDARY_SHIFT: n <-> n+1/n-1
    if line.contains(" + 1") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" + 1", " - 1", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
    } else if line.contains(" - 1") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" - 1", " + 1", 1),
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
    fn test_remove_safe_navigation() {
        let m = generate_ruby_line_mutants("role = user&.role", 10, Some("auth.rb"), 0);
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].operator, "REMOVE_SAFE_NAVIGATION");
        assert_eq!(m[0].mutated_line, "role = user.role");
    }

    #[test]
    fn test_invert_equality() {
        let m1 = generate_ruby_line_mutants("if role.eql?('admin')", 12, Some("auth.rb"), 0);
        assert_eq!(m1.len(), 1);
        assert_eq!(m1[0].operator, "INVERT_EQUALITY");
        assert_eq!(m1[0].mutated_line, "if !role.eql?('admin')");

        let m2 = generate_ruby_line_mutants("if status === 200", 14, Some("auth.rb"), 0);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "INVERT_EQUALITY");
        assert_eq!(m2[0].mutated_line, "if status !== 200");

        let m3 = generate_ruby_line_mutants("if count == 10", 16, Some("auth.rb"), 0);
        assert_eq!(m3.len(), 1);
        assert_eq!(m3[0].operator, "INVERT_EQUALITY");
        assert_eq!(m3[0].mutated_line, "if count != 10");
    }

    #[test]
    fn test_return_nil() {
        let m = generate_ruby_line_mutants("  return token", 20, Some("auth.rb"), 0);
        assert_eq!(m.len(), 1);
        assert_eq!(m[0].operator, "RETURN_NIL");
        assert_eq!(m[0].mutated_line, "  return nil");
    }

    #[test]
    fn test_boundary_shift() {
        let m1 = generate_ruby_line_mutants("if quota < max_quota", 30, Some("auth.rb"), 0);
        assert_eq!(m1.len(), 1);
        assert_eq!(m1[0].operator, "BOUNDARY_SHIFT");
        assert_eq!(m1[0].mutated_line, "if quota <= max_quota");

        let m2 = generate_ruby_line_mutants("next_step = step + 1", 32, Some("auth.rb"), 0);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "BOUNDARY_SHIFT");
        assert_eq!(m2[0].mutated_line, "next_step = step - 1");
    }
}
