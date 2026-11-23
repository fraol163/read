use super::Mutant;

/// Detects if a file path is a TypeScript or JavaScript source file.
pub fn is_ts_js_file(path: &str) -> bool {
    let lower = path.to_lowercase();
    lower.ends_with(".ts")
        || lower.ends_with(".tsx")
        || lower.ends_with(".js")
        || lower.ends_with(".jsx")
        || lower.ends_with(".mjs")
        || lower.ends_with(".cjs")
}

/// Generates targeted AST mutation candidates for a single diff line in TS/JS files.
/// Strictly implements:
/// 1. INVERT_EQUALITY (=== <-> !== and == <-> !=)
/// 2. RETURN_UNDEFINED (return expr -> return undefined;)
/// 3. BOUNDARY_SHIFT (+ 1 <-> - 1, ++ <-> --, relational comparators, boundary zeros)
/// 4. REMOVE_OPTIONAL_CHAIN (?. -> .)
pub fn generate_ts_js_line_mutants(
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

    // 1. INVERT_EQUALITY: === <-> !== and == <-> !=
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
    }

    if line.contains(" == ") && !line.contains(" === ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" == ", " != ", 1),
            line_number: line_num,
            operator: "INVERT_EQUALITY".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" != ") && !line.contains(" !== ") {
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

    // 2. RETURN_UNDEFINED: return <expr>; -> return undefined; (suppressed for boolean literal returns where FLIP_BOOL applies and undefined causes compile error under strictNullChecks)
    let is_bool_return = trimmed == "return true;"
        || trimmed == "return false;"
        || trimmed.starts_with("return true")
        || trimmed.starts_with("return false");

    if (trimmed.starts_with("return ") || trimmed == "return")
        && !trimmed.contains("return undefined")
        && !is_bool_return
    {
        let indent: String = line.chars().take_while(|c| c.is_whitespace()).collect();
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: format!("{}return undefined;", indent),
            line_number: line_num,
            operator: "RETURN_UNDEFINED".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 3. BOUNDARY_SHIFT: (+ 1 <-> - 1, ++ <-> --, relational comparators, boundary checks)
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
            mutated_line: line.replacen(" < ", " >= ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" > ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" > ", " <= ", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    if line.contains(" > 0") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" > 0", " >= 0", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains(" < 0") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" < 0", " <= 0", 1),
            line_number: line_num,
            operator: "BOUNDARY_SHIFT".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // 4. REMOVE_OPTIONAL_CHAIN: ?. -> .
    if line.contains("?.") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("?.", ".", 1),
            line_number: line_num,
            operator: "REMOVE_OPTIONAL_CHAIN".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    // Additional standard boolean / logical operators for JS/TS
    if line.contains("true") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("true", "false", 1),
            line_number: line_num,
            operator: "FLIP_BOOL_TRUE".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    } else if line.contains("false") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen("false", "true", 1),
            line_number: line_num,
            operator: "FLIP_BOOL_FALSE".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        next_id += 1;
    }

    if line.contains(" && ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" && ", " || ", 1),
            line_number: line_num,
            operator: "LOGICAL_AND_TO_OR".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        // next_id += 1;
    } else if line.contains(" || ") {
        candidates.push(Mutant {
            id: next_id,
            original_line: line.to_string(),
            mutated_line: line.replacen(" || ", " && ", 1),
            line_number: line_num,
            operator: "LOGICAL_OR_TO_AND".to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        // next_id += 1;
    }

    candidates
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_invert_equality() {
        let m1 = generate_ts_js_line_mutants("if (user.role === \"admin\") {", 10, Some("service.ts"), 0);
        assert_eq!(m1.len(), 1);
        assert_eq!(m1[0].operator, "INVERT_EQUALITY");
        assert_eq!(m1[0].mutated_line, "if (user.role !== \"admin\") {");

        let m2 = generate_ts_js_line_mutants("if (user.status !== 200) {", 11, Some("service.ts"), 1);
        assert_eq!(m2.len(), 1);
        assert_eq!(m2[0].operator, "INVERT_EQUALITY");
        assert_eq!(m2[0].mutated_line, "if (user.status === 200) {");

        let m3 = generate_ts_js_line_mutants("if (a == b) {", 12, Some("app.js"), 2);
        assert_eq!(m3.len(), 1);
        assert_eq!(m3[0].operator, "INVERT_EQUALITY");
        assert_eq!(m3[0].mutated_line, "if (a != b) {");
    }

    #[test]
    fn test_return_undefined() {
        let m = generate_ts_js_line_mutants("    return res.status(200).json({ ok: true });", 20, Some("api.ts"), 0);
        let return_mutant = m.iter().find(|x| x.operator == "RETURN_UNDEFINED").unwrap();
        assert_eq!(return_mutant.mutated_line, "    return undefined;");
    }

    #[test]
    fn test_boundary_shift() {
        let m1 = generate_ts_js_line_mutants("const next = count + 1;", 30, Some("counter.ts"), 0);
        assert!(m1.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("count - 1")));

        let m2 = generate_ts_js_line_mutants("counter++;", 31, Some("counter.ts"), 1);
        assert!(m2.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("counter--;")));

        let m3 = generate_ts_js_line_mutants("if (len > 0) {", 32, Some("counter.ts"), 2);
        assert!(m3.iter().any(|x| x.operator == "BOUNDARY_SHIFT" && x.mutated_line.contains("len >= 0")));
    }

    #[test]
    fn test_remove_optional_chain() {
        let m = generate_ts_js_line_mutants("const name = req.user?.profile?.name;", 40, Some("auth.ts"), 0);
        let opt_mutant = m.iter().find(|x| x.operator == "REMOVE_OPTIONAL_CHAIN").unwrap();
        assert_eq!(opt_mutant.mutated_line, "const name = req.user.profile?.name;");
    }
}
