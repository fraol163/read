use serde::{Deserialize, Serialize};
use std::path::Path;
use std::time::{Duration, Instant};

pub mod cpp;
pub mod csharp;
pub mod java;
pub mod ruby;
pub mod ts_js;

const DEFAULT_MAX_MUTANTS: usize = 10;
const DEFAULT_TIMEOUT_MILLIS: u64 = 4500; // < 4.5 seconds hard execution timeout
const MAX_DIFF_BYTES: usize = 50 * 1024 * 1024; // 50 MB ceiling to enforce < 1.2 GB RAM usage

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Mutant {
    pub id: usize,
    pub original_line: String,
    pub mutated_line: String,
    pub line_number: usize,
    pub operator: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_path: Option<String>,
}

pub const REQUIRED_PASS_THRESHOLD_PERCENT: f64 = 100.0;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct MutationScoreReport {
    pub total_mutants: usize,
    pub killed_mutants: usize,
    pub surviving_mutants: usize,
    pub timed_out_mutants: usize,
    pub kill_score_percentage: f64,
}

impl MutationScoreReport {
    pub fn new(total: usize, killed: usize, timed_out: usize) -> Self {
        let surviving = total.saturating_sub(killed + timed_out);
        let score = if total == 0 {
            100.0
        } else {
            ((killed + timed_out) as f64 / total as f64) * 100.0
        };
        Self {
            total_mutants: total,
            killed_mutants: killed,
            surviving_mutants: surviving,
            timed_out_mutants: timed_out,
            kill_score_percentage: score,
        }
    }

    /// Evaluates whether the mutation score strictly meets the 100.0% pass criteria.
    /// Strictly requires zero surviving mutants. Any surviving mutant is an immediate failure.
    pub fn passes_gate(&self) -> bool {
        self.surviving_mutants == 0 && (self.kill_score_percentage - REQUIRED_PASS_THRESHOLD_PERCENT).abs() < f64::EPSILON
    }

    /// Validates the gate and returns an error with surviving mutants if any survived.
    pub fn validate_gate(&self, surviving_mutants: &[Mutant]) -> Result<(), String> {
        if !self.passes_gate() {
            let mut msg = format!(
                "AROP Gate Failure: Mutation score {:.1}% is below strictly required 100.0% threshold (Killed {}/{}). {} mutant(s) survived:\n",
                self.kill_score_percentage,
                self.killed_mutants,
                self.total_mutants,
                self.surviving_mutants
            );
            for m in surviving_mutants {
                msg.push_str(&format!(
                    "  - [SURVIVOR] Mutant #{}: {} at {}:{} -> {}\n",
                    m.id,
                    m.operator,
                    m.file_path.as_deref().unwrap_or("unknown"),
                    m.line_number,
                    m.mutated_line.trim()
                ));
            }
            return Err(msg);
        }
        Ok(())
    }
}

/// Calculates exact mutation kill score.
pub fn calculate_mutation_score(killed: usize, total: usize) -> MutationScoreReport {
    MutationScoreReport::new(total, killed, 0)
}

pub struct BoundedMutator {
    pub max_mutants: usize,
    pub timeout_duration: Duration,
    pub max_diff_bytes: usize,
}

impl Default for BoundedMutator {
    fn default() -> Self {
        Self::new()
    }
}

/// Masks text inside string literals (single, double, triple quoted) and comments with spaces,
/// preserving exact byte length and character offsets for AST-safe operator scanning.
pub fn mask_strings_and_comments(line: &str) -> String {
    let bytes = line.as_bytes();
    let mut masked = bytes.to_vec();
    let len = bytes.len();
    let mut i = 0;

    enum State {
        Code,
        InSingleQuote,
        InDoubleQuote,
        InTripleSingleQuote,
        InTripleDoubleQuote,
        InBlockComment,
    }

    let mut state = State::Code;

    while i < len {
        match state {
            State::Code => {
                if i + 2 < len && &bytes[i..i + 3] == b"\"\"\"" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    masked[i + 2] = b' ';
                    i += 3;
                    state = State::InTripleDoubleQuote;
                } else if i + 2 < len && &bytes[i..i + 3] == b"'''" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    masked[i + 2] = b' ';
                    i += 3;
                    state = State::InTripleSingleQuote;
                } else if bytes[i] == b'"' {
                    masked[i] = b' ';
                    i += 1;
                    state = State::InDoubleQuote;
                } else if bytes[i] == b'\'' {
                    masked[i] = b' ';
                    i += 1;
                    state = State::InSingleQuote;
                } else if i + 1 < len && &bytes[i..i + 2] == b"//" {
                    for j in i..len {
                        masked[j] = b' ';
                    }
                    break;
                } else if bytes[i] == b'#' {
                    for j in i..len {
                        masked[j] = b' ';
                    }
                    break;
                } else if i + 1 < len && &bytes[i..i + 2] == b"/*" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    i += 2;
                    state = State::InBlockComment;
                } else {
                    i += 1;
                }
            }
            State::InDoubleQuote => {
                if bytes[i] == b'\\' {
                    masked[i] = b' ';
                    i += 1;
                    if i < len {
                        masked[i] = b' ';
                        i += 1;
                    }
                } else if bytes[i] == b'"' {
                    masked[i] = b' ';
                    i += 1;
                    state = State::Code;
                } else {
                    masked[i] = b' ';
                    i += 1;
                }
            }
            State::InSingleQuote => {
                if bytes[i] == b'\\' {
                    masked[i] = b' ';
                    i += 1;
                    if i < len {
                        masked[i] = b' ';
                        i += 1;
                    }
                } else if bytes[i] == b'\'' {
                    masked[i] = b' ';
                    i += 1;
                    state = State::Code;
                } else {
                    masked[i] = b' ';
                    i += 1;
                }
            }
            State::InTripleDoubleQuote => {
                if i + 2 < len && &bytes[i..i + 3] == b"\"\"\"" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    masked[i + 2] = b' ';
                    i += 3;
                    state = State::Code;
                } else {
                    masked[i] = b' ';
                    i += 1;
                }
            }
            State::InTripleSingleQuote => {
                if i + 2 < len && &bytes[i..i + 3] == b"'''" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    masked[i + 2] = b' ';
                    i += 3;
                    state = State::Code;
                } else {
                    masked[i] = b' ';
                    i += 1;
                }
            }
            State::InBlockComment => {
                if i + 1 < len && &bytes[i..i + 2] == b"*/" {
                    masked[i] = b' ';
                    masked[i + 1] = b' ';
                    i += 2;
                    state = State::Code;
                } else {
                    masked[i] = b' ';
                    i += 1;
                }
            }
        }
    }

    String::from_utf8(masked).unwrap_or_else(|_| line.to_string())
}

/// Applies a targeted mutation only if the target exists outside strings and comments.
pub fn apply_mutation(
    line: &str,
    masked: &str,
    target: &str,
    replacement: &str,
    operator: &str,
    line_num: usize,
    file_path: Option<&str>,
    next_id: &mut usize,
    candidates: &mut Vec<Mutant>,
) -> bool {
    if let Some(idx) = masked.find(target) {
        let mutated_line = format!("{}{}{}", &line[..idx], replacement, &line[idx + target.len()..]);
        candidates.push(Mutant {
            id: *next_id,
            original_line: line.to_string(),
            mutated_line,
            line_number: line_num,
            operator: operator.to_string(),
            file_path: file_path.map(|s| s.to_string()),
        });
        *next_id += 1;
        true
    } else {
        false
    }
}

impl BoundedMutator {
    pub fn new() -> Self {
        Self {
            max_mutants: DEFAULT_MAX_MUTANTS,
            timeout_duration: Duration::from_millis(DEFAULT_TIMEOUT_MILLIS),
            max_diff_bytes: MAX_DIFF_BYTES,
        }
    }

    /// Generates targeted mutants from the current `git diff --staged` output.
    pub fn generate_from_staged_diff<P: AsRef<Path>>(&self, repo_root: P) -> Result<Vec<Mutant>, String> {
        let (_, diff_text) = crate::hasher::DiffHasher::hash_staged_diff(repo_root.as_ref())?;
        Ok(self.generate_diff_mutants(&diff_text))
    }

    /// Generates a strictly bounded list of mutants (maximum 10) targeted ONLY at lines
    /// touched by the unified git diff across Python, Rust, Go, and TypeScript/JavaScript.
    /// Strictly guarantees < 4.5s execution time and < 1.2 GB RAM ceiling.
    pub fn generate_diff_mutants(&self, diff_text: &str) -> Vec<Mutant> {
        let start_time = Instant::now();

        // Enforce memory ceiling by capping processed diff bytes
        let bounded_diff = if diff_text.len() > self.max_diff_bytes {
            &diff_text[..self.max_diff_bytes]
        } else {
            diff_text
        };

        let mut mutants = Vec::with_capacity(self.max_mutants);
        let mut current_line_num = 0;
        let mut current_file: Option<String> = None;
        let mut in_py_docstring = false;

        for line in bounded_diff.lines() {
            // Hard timeout protection: prevent infinite loops or CPU exhaustion
            if start_time.elapsed() >= self.timeout_duration {
                break;
            }

            // Track target file path from unified diff header (e.g. +++ b/path/to/file.rs)
            if let Some(stripped) = line.strip_prefix("+++ ") {
                let file_part = stripped.trim();
                let clean_path = if let Some(stripped_b) = file_part.strip_prefix("b/") {
                    stripped_b
                } else if let Some(stripped_i) = file_part.strip_prefix("i/") {
                    stripped_i
                } else if let Some(stripped_w) = file_part.strip_prefix("w/") {
                    stripped_w
                } else if file_part.len() > 2 && file_part.as_bytes()[1] == b'/' {
                    &file_part[2..]
                } else {
                    file_part
                };
                if clean_path != "/dev/null" {
                    current_file = Some(clean_path.to_string());
                    in_py_docstring = false;
                }
                continue;
            }

            // Track hunk header for line numbers (e.g. @@ -1,4 +10,5 @@)
            if line.starts_with("@@") {
                in_py_docstring = false;
                if let Some(plus_idx) = line.find('+') {
                    let sub = &line[plus_idx + 1..];
                    let comma_or_space = sub.find([',', ' ']).unwrap_or(sub.len());
                    if let Ok(num) = sub[..comma_or_space].parse::<usize>() {
                        current_line_num = num;
                    }
                }
                continue;
            }

            // Track Python docstring delimiters to avoid mutating text within docstrings
            let is_py = current_file.as_deref().map_or(false, |f| f.ends_with(".py"));
            if is_py && !line.starts_with("---") && !line.starts_with("+++") && !line.starts_with("@@") {
                let check_line = if line.starts_with('+') || line.starts_with('-') || line.starts_with(' ') {
                    &line[1..]
                } else {
                    line
                };
                let trip_count = check_line.matches("\"\"\"").count() + check_line.matches("'''").count();
                if in_py_docstring {
                    if trip_count % 2 != 0 {
                        in_py_docstring = false;
                    }
                    if line.starts_with('+') && !line.starts_with("+++") {
                        current_line_num += 1;
                    } else if !line.starts_with('-') {
                        current_line_num += 1;
                    }
                    continue;
                } else if trip_count % 2 != 0 {
                    in_py_docstring = true;
                    if line.starts_with('+') && !line.starts_with("+++") {
                        current_line_num += 1;
                    } else if !line.starts_with('-') {
                        current_line_num += 1;
                    }
                    continue;
                } else if trip_count >= 2 {
                    if line.starts_with('+') && !line.starts_with("+++") {
                        current_line_num += 1;
                    } else if !line.starts_with('-') {
                        current_line_num += 1;
                    }
                    continue;
                }
            }

            // Only mutate added/modified lines in the diff (starting with '+' but not '+++')
            if line.starts_with('+') && !line.starts_with("+++") {
                let code_line = &line[1..];
                let line_candidates = self.generate_line_mutants(
                    code_line,
                    current_line_num,
                    current_file.as_deref(),
                    mutants.len(),
                );

                for m in line_candidates {
                    mutants.push(m);
                    if mutants.len() >= self.max_mutants {
                        return mutants;
                    }
                }

                current_line_num += 1;
            } else if !line.starts_with('-') {
                current_line_num += 1;
            }
        }

        mutants
    }

    /// Evaluates all targeted mutation operators for a single modified line.
    fn generate_line_mutants(
        &self,
        line: &str,
        line_num: usize,
        file_path: Option<&str>,
        current_count: usize,
    ) -> Vec<Mutant> {
        // Fast path for TypeScript / JavaScript source files
        if let Some(path) = file_path {
            if path.starts_with(".read/")
                || path.starts_with(".method/")
                || path.ends_with(".log")
                || path.ends_with(".md")
                || path.ends_with(".json")
                || path.ends_with(".txt")
            {
                return Vec::new();
            }
            if ts_js::is_ts_js_file(path) {
                return ts_js::generate_ts_js_line_mutants(line, line_num, file_path, current_count);
            }
            if java::is_java_file(path) {
                return java::generate_java_line_mutants(line, line_num, file_path, current_count);
            }
            if csharp::is_csharp_file(path) {
                return csharp::generate_csharp_line_mutants(line, line_num, file_path, current_count);
            }
            if cpp::is_cpp_file(path) {
                return cpp::generate_cpp_line_mutants(line, line_num, file_path, current_count);
            }
            if ruby::is_ruby_file(path) {
                return ruby::generate_ruby_line_mutants(line, line_num, file_path, current_count);
            }
        }

        let mut candidates = Vec::new();
        let mut next_id = current_count + 1;

        // Skip comments and blank lines
        let trimmed = line.trim();
        if trimmed.is_empty()
            || trimmed.starts_with("//")
            || trimmed.starts_with('#')
            || trimmed.starts_with("/*")
            || trimmed.starts_with('*')
            || trimmed.starts_with("\"\"\"")
            || trimmed.starts_with("'''")
        {
            return candidates;
        }

        let masked = mask_strings_and_comments(line);
        if masked.trim().is_empty() {
            return candidates;
        }

        // --------------------------------------------------------------------
        // Operator Group 1: Relational & Equality Operator Flips
        // --------------------------------------------------------------------
        // JS/TS strict equality flips
        if !apply_mutation(line, &masked, " === ", " !== ", "STRICT_EQ_TO_NEQ", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, " !== ", " === ", "STRICT_NEQ_TO_EQ", line_num, file_path, &mut next_id, &mut candidates);
        }

        // Standard equality flips (Python, Rust, Go, JS)
        if !masked.contains(" === ") && !masked.contains(" !== ") {
            if !apply_mutation(line, &masked, " == ", " != ", "INVERT_EQ", line_num, file_path, &mut next_id, &mut candidates) {
                apply_mutation(line, &masked, " != ", " == ", "INVERT_NEQ", line_num, file_path, &mut next_id, &mut candidates);
            }
        }

        // Relational comparators (<, >, <=, >=)
        if !apply_mutation(line, &masked, " <= ", " > ", "INVERT_LE", line_num, file_path, &mut next_id, &mut candidates) {
            if !apply_mutation(line, &masked, " >= ", " < ", "INVERT_GE", line_num, file_path, &mut next_id, &mut candidates) {
                if !apply_mutation(line, &masked, " < ", " >= ", "INVERT_LT", line_num, file_path, &mut next_id, &mut candidates) {
                    apply_mutation(line, &masked, " > ", " <= ", "INVERT_GT", line_num, file_path, &mut next_id, &mut candidates);
                }
            }
        }

        // --------------------------------------------------------------------
        // Operator Group 2: Boolean Literal & Logical Operator Flips
        // --------------------------------------------------------------------
        if !apply_mutation(line, &masked, "True", "False", "FLIP_BOOL_TRUE", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, "False", "True", "FLIP_BOOL_FALSE", line_num, file_path, &mut next_id, &mut candidates);
        }

        if !apply_mutation(line, &masked, "true", "false", "FLIP_BOOL_TRUE", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, "false", "true", "FLIP_BOOL_FALSE", line_num, file_path, &mut next_id, &mut candidates);
        }

        // Logical AND / OR
        if !apply_mutation(line, &masked, " && ", " || ", "LOGICAL_AND_TO_OR", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, " || ", " && ", "LOGICAL_OR_TO_AND", line_num, file_path, &mut next_id, &mut candidates);
        }

        if !apply_mutation(line, &masked, " and ", " or ", "LOGICAL_AND_TO_OR", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, " or ", " and ", "LOGICAL_OR_TO_AND", line_num, file_path, &mut next_id, &mut candidates);
        }

        // --------------------------------------------------------------------
        // Operator Group 3: Boundary Offsets
        // --------------------------------------------------------------------
        if !apply_mutation(line, &masked, " + 1", " - 1", "OFF_BY_ONE_ADD", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, " - 1", " + 1", "OFF_BY_ONE_SUB", line_num, file_path, &mut next_id, &mut candidates);
        }

        if !apply_mutation(line, &masked, "++", "--", "BOUNDARY_INC", line_num, file_path, &mut next_id, &mut candidates) {
            if !line.starts_with("---") {
                apply_mutation(line, &masked, "--", "++", "BOUNDARY_DEC", line_num, file_path, &mut next_id, &mut candidates);
            }
        }

        if !apply_mutation(line, &masked, " > 0", " >= 0", "BOUNDARY_GT_ZERO", line_num, file_path, &mut next_id, &mut candidates) {
            apply_mutation(line, &masked, " < 0", " <= 0", "BOUNDARY_LT_ZERO", line_num, file_path, &mut next_id, &mut candidates);
        }

        // --------------------------------------------------------------------
        // Operator Group 4: Return Value Substitutions Across Languages
        // --------------------------------------------------------------------
        if trimmed.starts_with("return ") || trimmed == "return" {
            let indent: String = line.chars().take_while(|c| c.is_whitespace()).collect();
            let ext = file_path
                .and_then(|p| p.split('.').next_back())
                .unwrap_or("");

            match ext {
                "go" => {
                    if !trimmed.ends_with("nil") {
                        candidates.push(Mutant {
                            id: next_id,
                            original_line: line.to_string(),
                            mutated_line: format!("{}return nil", indent),
                            line_number: line_num,
                            operator: "RETURN_NIL".to_string(),
                            file_path: file_path.map(|s| s.to_string()),
                        });
                    }
                }
                "ts" | "tsx" | "js" | "jsx" => {
                    if !trimmed.ends_with("null") && !trimmed.ends_with("undefined") {
                        candidates.push(Mutant {
                            id: next_id,
                            original_line: line.to_string(),
                            mutated_line: format!("{}return null;", indent),
                            line_number: line_num,
                            operator: "RETURN_NULL".to_string(),
                            file_path: file_path.map(|s| s.to_string()),
                        });
                    }
                }
                "rs" => {
                    if !trimmed.contains("Default::default()") && !trimmed.contains("None") {
                        candidates.push(Mutant {
                            id: next_id,
                            original_line: line.to_string(),
                            mutated_line: format!("{}return Ok(Default::default());", indent),
                            line_number: line_num,
                            operator: "RETURN_DEFAULT_RUST".to_string(),
                            file_path: file_path.map(|s| s.to_string()),
                        });
                    }
                }
                _ => {
                    // Python and generic fallback
                    if !trimmed.ends_with("None") {
                        candidates.push(Mutant {
                            id: next_id,
                            original_line: line.to_string(),
                            mutated_line: format!("{}return None", indent),
                            line_number: line_num,
                            operator: "RETURN_NONE".to_string(),
                            file_path: file_path.map(|s| s.to_string()),
                        });
                    }
                }
            }
        }

        candidates
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_diff_targeted_mutator() {
        let diff = r#"
--- a/auth_service.py
+++ b/auth_service.py
@@ -10,4 +10,6 @@
 def check_access(user):
-    return False
+    if user.is_admin == True:
+        return True + 1
+    return False
"#;

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);

        assert!(!mutants.is_empty());
        assert!(mutants.len() <= 10);

        let ops: Vec<&str> = mutants.iter().map(|m| m.operator.as_str()).collect();
        assert!(ops.contains(&"INVERT_EQ") || ops.contains(&"FLIP_BOOL_TRUE") || ops.contains(&"OFF_BY_ONE_ADD"));
    }

    #[test]
    fn test_rust_diff_mutator() {
        let diff = r#"
--- a/src/service.rs
+++ b/src/service.rs
@@ -20,5 +20,7 @@
 pub fn authenticate(user: &User) -> Result<bool, AuthError> {
-    Ok(false)
+    if user.role == "admin" && user.is_active == true {
+        let quota = user.quota + 1;
+        return Ok(quota > 0);
+    }
     Ok(false)
 }
"#;

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);

        assert!(!mutants.is_empty());
        assert!(mutants.len() <= 10);

        let ops: Vec<&str> = mutants.iter().map(|m| m.operator.as_str()).collect();
        assert!(ops.contains(&"INVERT_EQ") || ops.contains(&"FLIP_BOOL_TRUE") || ops.contains(&"LOGICAL_AND_TO_OR") || ops.contains(&"OFF_BY_ONE_ADD"));
    }

    #[test]
    fn test_go_diff_mutator() {
        let diff = r#"
--- a/auth.go
+++ b/auth.go
@@ -15,4 +15,6 @@
 func Validate(token string) error {
-    return nil
+    if token != "" && len(token) > 0 {
+        return errors.New("invalid")
+    }
     return nil
 }
"#;

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);

        assert!(!mutants.is_empty());
        assert!(mutants.len() <= 10);

        let ops: Vec<&str> = mutants.iter().map(|m| m.operator.as_str()).collect();
        assert!(ops.contains(&"INVERT_NEQ") || ops.contains(&"LOGICAL_AND_TO_OR") || ops.contains(&"BOUNDARY_GT_ZERO") || ops.contains(&"RETURN_NIL"));
    }

    #[test]
    fn test_typescript_diff_mutator() {
        let diff = r#"
--- a/auth.ts
+++ b/auth.ts
@@ -12,4 +12,6 @@
 export function checkPermission(role: string): boolean {
-    return false;
+    if (role === "admin" || role === "super") {
+        return true;
+    }
     return false;
 }
"#;

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);

        assert!(!mutants.is_empty());
        assert!(mutants.len() <= 10);

        let ops: Vec<&str> = mutants.iter().map(|m| m.operator.as_str()).collect();
        assert!(ops.contains(&"INVERT_EQUALITY") || ops.contains(&"LOGICAL_OR_TO_AND") || ops.contains(&"FLIP_BOOL_TRUE") || ops.contains(&"RETURN_UNDEFINED"));
    }

    #[test]
    fn test_java_diff_mutator() {
        let diff = r#"
--- a/AuthService.java
+++ b/AuthService.java
@@ -15,5 +15,7 @@
 public boolean verify(User user) {
-    return false;
+    if (user != null && user.getRole().equals("ADMIN")) {
+        int quota = user.getQuota() + 1;
+        return quota > 0;
+    }
     return false;
 }
"#;

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);

        assert!(!mutants.is_empty());
        assert!(mutants.len() <= 10);

        let ops: Vec<&str> = mutants.iter().map(|m| m.operator.as_str()).collect();
        assert!(ops.contains(&"REMOVE_NULL_CHECK") || ops.contains(&"INVERT_EQUALITY") || ops.contains(&"BOUNDARY_SHIFT") || ops.contains(&"RETURN_NULL"));
    }

    #[test]
    fn test_bounded_mutant_cap() {
        // Generate a large diff with 50 lines capable of mutation
        let mut diff = String::from("--- a/big.py\n+++ b/big.py\n@@ -1,50 +1,50 @@\n");
        for i in 0..50 {
            diff.push_str(&format!("+    x_{} = a == b and True + 1\n", i));
        }

        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(&diff);

        // Strictly bounded to maximum 10 mutants
        assert_eq!(mutants.len(), 10);
    }

    #[test]
    fn test_mutation_score_calculation() {
        let report_zero = calculate_mutation_score(0, 10);
        assert_eq!(report_zero.total_mutants, 10);
        assert_eq!(report_zero.killed_mutants, 0);
        assert_eq!(report_zero.surviving_mutants, 10);
        assert!((report_zero.kill_score_percentage - 0.0).abs() < f64::EPSILON);

        let report_partial = calculate_mutation_score(8, 10);
        assert_eq!(report_partial.total_mutants, 10);
        assert_eq!(report_partial.killed_mutants, 8);
        assert_eq!(report_partial.surviving_mutants, 2);
        assert!((report_partial.kill_score_percentage - 80.0).abs() < f64::EPSILON);
        assert!(!report_partial.passes_gate());

        let report_ninety = calculate_mutation_score(9, 10);
        assert_eq!(report_ninety.total_mutants, 10);
        assert_eq!(report_ninety.killed_mutants, 9);
        assert_eq!(report_ninety.surviving_mutants, 1);
        assert!((report_ninety.kill_score_percentage - 90.0).abs() < f64::EPSILON);
        assert!(!report_ninety.passes_gate(), "90% mutation score MUST be rejected by strictly 100% gate");

        let report_full = calculate_mutation_score(10, 10);
        assert_eq!(report_full.total_mutants, 10);
        assert_eq!(report_full.killed_mutants, 10);
        assert_eq!(report_full.surviving_mutants, 0);
        assert!((report_full.kill_score_percentage - 100.0).abs() < f64::EPSILON);
        assert!(report_full.passes_gate());

        let report_empty = calculate_mutation_score(0, 0);
        assert_eq!(report_empty.total_mutants, 0);
        assert!((report_empty.kill_score_percentage - 100.0).abs() < f64::EPSILON);
        assert!(report_empty.passes_gate());
    }

    #[test]
    fn test_gate_validation_error_formatting() {
        let report_survivor = calculate_mutation_score(9, 10);
        let surviving = vec![Mutant {
            id: 4,
            original_line: "if user.role == \"admin\":".to_string(),
            mutated_line: "if user.role != \"admin\":".to_string(),
            line_number: 42,
            operator: "INVERT_EQ".to_string(),
            file_path: Some("auth.py".to_string()),
        }];

        let res = report_survivor.validate_gate(&surviving);
        assert!(res.is_err());
        let err = res.unwrap_err();
        assert!(err.contains("AROP Gate Failure"));
        assert!(err.contains("90.0%"));
        assert!(err.contains("Mutant #4: INVERT_EQ at auth.py:42"));
    }

    #[test]
    fn test_csharp_diff_mutator() {
        let diff = r#"
--- a/TokenService.cs
+++ b/TokenService.cs
@@ -10,6 +10,6 @@
+    var user = auth?.User;
+    if (user.Role.Equals("ADMIN")) {
+        return true;
+    }
+    return false;
"#;
        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);
        assert!(!mutants.is_empty());
        assert!(mutants.iter().any(|m| m.operator == "REMOVE_NULL_CONDITIONAL"));
        assert!(mutants.iter().any(|m| m.operator == "INVERT_EQUALITY"));
        assert!(mutants.iter().any(|m| m.operator == "RETURN_DEFAULT"));
    }

    #[test]
    fn test_cpp_diff_mutator() {
        let diff = r#"
--- a/Buffer.cpp
+++ b/Buffer.cpp
@@ -10,6 +10,6 @@
+    if (ptr != nullptr) {
+        if (size <= max_size) {
+            return ptr;
+        }
+    }
+    return nullptr;
"#;
        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);
        assert!(!mutants.is_empty());
        assert!(mutants.iter().any(|m| m.operator == "REMOVE_NULLPTR_CHECK"));
        assert!(mutants.iter().any(|m| m.operator == "INVERT_EQUALITY"));
        assert!(mutants.iter().any(|m| m.operator == "RETURN_DEFAULT"));
    }

    #[test]
    fn test_diff_mutator_skips_python_docstrings() {
        let diff = r#"
--- a/service.py
+++ b/service.py
@@ -1,15 +1,15 @@
+def process_payload(data):
+    """
+    This is a docstring with == and != and or and and keywords.
+    It should not produce any mutants.
+    """
+    '''
+    Another docstring with return None or return False
+    '''
+    """Single line docstring with x == y"""
+    if data == "admin":
+        return True
+    return False
+"#;
        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);
        assert!(!mutants.is_empty());
        // Verify no mutant was produced from inside docstrings
        for m in &mutants {
            assert!(
                !m.original_line.contains("docstring"),
                "Mutant generated from docstring: {:?}",
                m
            );
        }
        // Verify code mutants were generated
        assert!(mutants.iter().any(|m| m.original_line.contains("data == \"admin\"")));
    }

    #[test]
    fn test_ruby_diff_mutator() {
        let diff = r#"
--- a/auth.rb
+++ b/auth.rb
@@ -10,6 +10,6 @@
+    role = user&.role
+    if role.eql?('admin')
+        return true
+    end
+    return false
"#;
        let mutator = BoundedMutator::new();
        let mutants = mutator.generate_diff_mutants(diff);
        assert!(!mutants.is_empty());
        assert!(mutants.iter().any(|m| m.operator == "REMOVE_SAFE_NAVIGATION"));
        assert!(mutants.iter().any(|m| m.operator == "INVERT_EQUALITY"));
        assert!(mutants.iter().any(|m| m.operator == "RETURN_NIL"));
    }

    #[test]
    fn test_execution_timeout_and_memory_ceiling() {
        // Enforce execution timeout constraint (< 4.5s)
        let start = Instant::now();
        let mutator = BoundedMutator::new();
        let large_diff = "--- a/large.py\n+++ b/large.py\n@@ -1,100 +1,100 @@\n".to_string()
            + &"+    value = test == 42\n".repeat(500);

        let mutants = mutator.generate_diff_mutants(&large_diff);
        let elapsed = start.elapsed();

        assert_eq!(mutants.len(), 10);
        assert!(elapsed < Duration::from_millis(4500));
    }

    #[test]
    fn test_mutant_on_line_whose_only_operator_is_inside_string_literal_not_generated() {
        let mutator = BoundedMutator::new();

        // Double quoted string literal with logical operator "and"
        let line_double = r#"raise ValueError("page and page_size must be positive integers")"#;
        let mutants_double = mutator.generate_line_mutants(line_double, 6, Some("paginate.py"), 0);
        assert!(
            mutants_double.is_empty(),
            "Expected no mutants on double-quoted string literal with operator, got: {:?}",
            mutants_double
        );

        // Single quoted string literal with comparison operator " == " and " or "
        let line_single = r#"msg = 'status == active or count + 1'"#;
        let mutants_single = mutator.generate_line_mutants(line_single, 1, Some("test.py"), 0);
        assert!(
            mutants_single.is_empty(),
            "Expected no mutants on single-quoted string literal with operator, got: {:?}",
            mutants_single
        );

        // Triple double quoted string literal with " == " and " != "
        let line_triple_double = r#"doc = """check if x == y and z != 0""""#;
        let mutants_triple_double = mutator.generate_line_mutants(line_triple_double, 1, Some("test.py"), 0);
        assert!(
            mutants_triple_double.is_empty(),
            "Expected no mutants on triple-double-quoted string literal with operator, got: {:?}",
            mutants_triple_double
        );

        // Triple single quoted string literal with " or " and " + 1"
        let line_triple_single = r#"doc = '''retry or attempt + 1'''"#;
        let mutants_triple_single = mutator.generate_line_mutants(line_triple_single, 1, Some("test.py"), 0);
        assert!(
            mutants_triple_single.is_empty(),
            "Expected no mutants on triple-single-quoted string literal with operator, got: {:?}",
            mutants_triple_single
        );

        // Full diff test showing diff with string literal lines only produces zero mutants
        let diff = r#"
--- a/test.py
+++ b/test.py
@@ -1,4 +1,4 @@
+raise ValueError("page and page_size must be positive integers")
+error_msg = "user == admin or role == root"
+summary = 'count + 1 and offset - 1'
+docstring = """status == active and retry or timeout"""
"#;
        let mutants = mutator.generate_diff_mutants(diff);
        assert!(
            mutants.is_empty(),
            "Expected no mutants generated for lines with operators exclusively in string literals, got: {:?}",
            mutants
        );
    }
}
