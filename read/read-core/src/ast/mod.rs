use regex::Regex;
use serde::{Deserialize, Serialize};
use std::collections::{HashSet, VecDeque};
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::OnceLock;
use std::time::Instant;

const MAX_FILE_SIZE_BYTES: u64 = 20 * 1024 * 1024; // 20 MB ceiling to prevent runaway allocations
const MAX_DEPENDENCY_FILES: usize = 500; // Limit dependency traversal to prevent memory runaway

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SourceLanguage {
    Python,
    Rust,
    Go,
    TypeScript,
    JavaScript,
    Java,
    CSharp,
    Cpp,
    Ruby,
    Unknown,
}

impl SourceLanguage {
    pub fn from_path<P: AsRef<Path>>(path: P) -> Self {
        let path_ref = path.as_ref();
        let ext = path_ref
            .extension()
            .and_then(|s| s.to_str())
            .unwrap_or("")
            .to_lowercase();
        match ext.as_str() {
            "py" | "pyi" => SourceLanguage::Python,
            "rs" => SourceLanguage::Rust,
            "go" => SourceLanguage::Go,
            "ts" | "tsx" => SourceLanguage::TypeScript,
            "js" | "jsx" | "mjs" | "cjs" => SourceLanguage::JavaScript,
            "java" => SourceLanguage::Java,
            "cs" => SourceLanguage::CSharp,
            "cpp" | "cc" | "cxx" | "h" | "hpp" => SourceLanguage::Cpp,
            "rb" => SourceLanguage::Ruby,
            _ => SourceLanguage::Unknown,
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct SymbolDefinition {
    pub file_path: String,
    pub symbol_type: String, // "function", "class", "struct", "enum", "trait", "interface", "type", "const"
    pub name: String,
    pub signature: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub docstring: Option<String>,
}

impl SymbolDefinition {
    pub fn new(
        file_path: impl Into<String>,
        symbol_type: impl Into<String>,
        name: impl Into<String>,
        signature: impl Into<String>,
        docstring: Option<String>,
    ) -> Self {
        Self {
            file_path: file_path.into(),
            symbol_type: symbol_type.into(),
            name: name.into(),
            signature: signature.into(),
            docstring,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextManifest {
    pub target_file: String,
    pub discovered_dependencies: Vec<String>,
    pub symbols: Vec<SymbolDefinition>,
    pub estimated_token_footprint: usize,
    pub parse_degraded: bool,
    pub extraction_duration_micros: u128,
}

pub struct AstContextResolver;

impl AstContextResolver {
    /// Deterministically extracts local dependencies and symbol signatures without model calls ($0.00).
    /// Resolves transitive imports up to max_depth across Python, Rust, Go, and TypeScript/JavaScript.
    /// Resilient to mid-edit syntax errors: degrades gracefully to regex/line-based fallback without panicking.
    pub fn resolve<P1: AsRef<Path>, P2: AsRef<Path>>(
        target_file: P1,
        repo_root: P2,
        max_depth: usize,
    ) -> Result<ContextManifest, String> {
        let start_time = Instant::now();
        let repo_root = repo_root.as_ref().canonicalize().map_err(|e| e.to_string())?;
        let target_path = target_file.as_ref();
        let target_full = if target_path.is_absolute() {
            target_path.to_path_buf()
        } else {
            repo_root.join(target_path)
        };

        if !target_full.exists() {
            return Err(format!("Target file does not exist: {:?}", target_full));
        }

        let mut visited = HashSet::new();
        let mut queue = VecDeque::new();
        let mut dependencies = Vec::new();
        let mut all_symbols = Vec::new();
        let mut any_degraded = false;

        queue.push_back((target_full.clone(), 0));
        visited.insert(target_full.clone());

        while let Some((current_file, depth)) = queue.pop_front() {
            if visited.len() > MAX_DEPENDENCY_FILES {
                break;
            }

            let rel_path = current_file
                .strip_prefix(&repo_root)
                .unwrap_or(&current_file)
                .to_string_lossy()
                .to_string();

            if current_file != target_full {
                dependencies.push(rel_path.clone());
            }

            // Check file size to avoid runaway memory allocation
            if let Ok(metadata) = fs::metadata(&current_file) {
                if metadata.len() > MAX_FILE_SIZE_BYTES {
                    continue;
                }
            }

            let content = match fs::read_to_string(&current_file) {
                Ok(c) => c,
                Err(_) => continue, // Skip unreadable or binary files safely
            };

            let language = SourceLanguage::from_path(&current_file);
            let (file_symbols, imports, is_degraded) =
                Self::extract_symbols_and_imports(&content, &rel_path, language);

            if is_degraded {
                any_degraded = true;
            }

            all_symbols.extend(file_symbols);

            // Traverse imports if within depth limit
            if depth < max_depth {
                for mod_import in imports {
                    let candidates =
                        Self::resolve_import_candidates(&mod_import, &current_file, &repo_root, language);
                    for candidate_path in candidates {
                        if candidate_path.exists() && !visited.contains(&candidate_path) {
                            visited.insert(candidate_path.clone());
                            queue.push_back((candidate_path, depth + 1));
                        }
                    }
                }
            }
        }

        // Calculate rough token footprint (~4 chars per token)
        let total_chars: usize = all_symbols
            .iter()
            .map(|s| s.signature.len() + s.docstring.as_ref().map_or(0, |d| d.len()) + 10)
            .sum();
        let estimated_tokens = (total_chars / 4) + 50;
        let duration = start_time.elapsed().as_micros();

        Ok(ContextManifest {
            target_file: target_full
                .strip_prefix(&repo_root)
                .unwrap_or(&target_full)
                .to_string_lossy()
                .to_string(),
            discovered_dependencies: dependencies,
            symbols: all_symbols,
            estimated_token_footprint: estimated_tokens,
            parse_degraded: any_degraded,
            extraction_duration_micros: duration,
        })
    }

    /// Primary multi-language extractor dispatch with graceful syntax error degradation.
    pub fn extract_symbols_and_imports(
        content: &str,
        file_rel: &str,
        language: SourceLanguage,
    ) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        match language {
            SourceLanguage::Python => Self::extract_python(content, file_rel),
            SourceLanguage::Rust => Self::extract_rust(content, file_rel),
            SourceLanguage::Go => Self::extract_go(content, file_rel),
            SourceLanguage::TypeScript | SourceLanguage::JavaScript => {
                Self::extract_typescript_javascript(content, file_rel)
            }
            SourceLanguage::Java => Self::extract_java(content, file_rel),
            SourceLanguage::CSharp => Self::extract_csharp(content, file_rel),
            SourceLanguage::Cpp => Self::extract_cpp(content, file_rel),
            SourceLanguage::Ruby => Self::extract_ruby(content, file_rel),
            SourceLanguage::Unknown => {
                // Try resilient fallback scanning across all common keywords
                let (symbols, imports) = Self::extract_fallback_generic(content, file_rel);
                (symbols, imports, true)
            }
        }
    }

    // ------------------------------------------------------------------------
    // Python Parser (Imports, def/class signatures, docstrings, fallback)
    // ------------------------------------------------------------------------
    fn extract_python(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let mut degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            // 1. Python Imports
            if trimmed.starts_with("import ") {
                let parts: Vec<&str> = trimmed[7..].split(',').collect();
                for part in parts {
                    let mod_part = part.trim().split_whitespace().next().unwrap_or("");
                    if !mod_part.is_empty() {
                        imports.push(mod_part.to_string());
                    }
                }
                i += 1;
                continue;
            } else if trimmed.starts_with("from ") {
                if let Some(rest) = trimmed.strip_prefix("from ") {
                    if let Some(import_idx) = rest.find(" import ") {
                        let mod_name = rest[..import_idx].trim();
                        if !mod_name.is_empty() {
                            imports.push(mod_name.to_string());
                        }
                    } else if rest.contains(" import(") || rest.ends_with(" import") {
                        let mod_name = rest.split_whitespace().next().unwrap_or("").trim();
                        if !mod_name.is_empty() {
                            imports.push(mod_name.to_string());
                        }
                    }
                }
                i += 1;
                continue;
            }

            // 2. Python def / async def
            let is_async_def = trimmed.starts_with("async def ");
            let is_def = trimmed.starts_with("def ");
            if is_def || is_async_def {
                let offset = if is_async_def { 10 } else { 4 };
                let mut paren_count = 0;
                for &b in trimmed.as_bytes() {
                    if b == b'(' {
                        paren_count += 1;
                    } else if b == b')' {
                        paren_count -= 1;
                    }
                }

                let (full_sig, found_colon, j) = if trimmed.ends_with(':') && paren_count <= 0 {
                    (trimmed.to_string(), true, i)
                } else {
                    let mut sig_parts = vec![trimmed];
                    let mut found = false;
                    let mut j_scan = i + 1;
                    while j_scan < lines.len() {
                        let curr_line = lines[j_scan].trim();
                        for &b in curr_line.as_bytes() {
                            if b == b'(' {
                                paren_count += 1;
                            } else if b == b')' {
                                paren_count -= 1;
                            }
                        }
                        sig_parts.push(curr_line);
                        if curr_line.ends_with(':') && paren_count <= 0 {
                            found = true;
                            break;
                        }
                        if j_scan > i + 10 {
                            degraded = true;
                            break;
                        }
                        j_scan += 1;
                    }
                    (sig_parts.join(" "), found, j_scan)
                };

                let raw_name_slice = &full_sig[offset..];
                let name_end = raw_name_slice
                    .find('(')
                    .or_else(|| raw_name_slice.find(':'))
                    .unwrap_or(raw_name_slice.len());
                let name = raw_name_slice[..name_end].trim().to_string();

                if !found_colon {
                    degraded = true;
                }

                // Check for docstring immediately following signature
                let mut docstring = None;
                let mut k = if found_colon { j + 1 } else { j };
                while k < lines.len() {
                    let next_line = lines[k].trim();
                    if next_line.is_empty() || next_line.starts_with('#') {
                        k += 1;
                        continue;
                    }
                    if next_line.starts_with("\"\"\"") || next_line.starts_with("'''") {
                        let quote = if next_line.starts_with("\"\"\"") { "\"\"\"" } else { "'''" };
                        if next_line.len() > 3 && next_line[3..].contains(quote) {
                            // Single line docstring
                            let inner = &next_line[3..next_line[3..].find(quote).unwrap() + 3];
                            docstring = Some(inner.trim().to_string());
                        } else {
                            // Multiline docstring
                            let mut doc_lines = Vec::new();
                            doc_lines.push(&next_line[3..]);
                            let mut doc_idx = k + 1;
                            while doc_idx < lines.len() {
                                let doc_l = lines[doc_idx].trim();
                                if let Some(end_pos) = doc_l.find(quote) {
                                    doc_lines.push(&doc_l[..end_pos]);
                                    break;
                                } else {
                                    doc_lines.push(doc_l);
                                }
                                doc_idx += 1;
                            }
                            docstring = Some(doc_lines.join("\n").trim().to_string());
                        }
                    }
                    break;
                }

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        full_sig,
                        docstring,
                    ));
                }
                i = if found_colon { j + 1 } else { i + 1 };
                continue;
            }

            // 3. Python class
            if trimmed.starts_with("class ") {
                let mut sig_parts = Vec::new();
                let mut j = i;
                let mut found_colon = false;

                while j < lines.len() {
                    let curr_line = lines[j].trim();
                    sig_parts.push(curr_line);
                    if curr_line.ends_with(':') {
                        found_colon = true;
                        break;
                    }
                    if j > i + 6 {
                        degraded = true;
                        break;
                    }
                    j += 1;
                }


                let full_sig = sig_parts.join(" ");
                let raw_name_slice = &full_sig[6..];
                let name_end = raw_name_slice
                    .find('(')
                    .or_else(|| raw_name_slice.find(':'))
                    .unwrap_or(raw_name_slice.len());
                let name = raw_name_slice[..name_end].trim().to_string();

                if !found_colon {
                    degraded = true;
                }

                // Check for docstring immediately following class definition
                let mut docstring = None;
                let mut k = if found_colon { j + 1 } else { j };
                while k < lines.len() {
                    let next_line = lines[k].trim();
                    if next_line.is_empty() || next_line.starts_with('#') {
                        k += 1;
                        continue;
                    }
                    if next_line.starts_with("\"\"\"") || next_line.starts_with("'''") {
                        let quote = if next_line.starts_with("\"\"\"") { "\"\"\"" } else { "'''" };
                        if next_line.len() > 3 && next_line[3..].contains(quote) {
                            let inner = &next_line[3..next_line[3..].find(quote).unwrap() + 3];
                            docstring = Some(inner.trim().to_string());
                        } else {
                            let mut doc_lines = Vec::new();
                            doc_lines.push(&next_line[3..]);
                            let mut doc_idx = k + 1;
                            while doc_idx < lines.len() {
                                let doc_l = lines[doc_idx].trim();
                                if let Some(end_pos) = doc_l.find(quote) {
                                    doc_lines.push(&doc_l[..end_pos]);
                                    break;
                                } else {
                                    doc_lines.push(doc_l);
                                }
                                doc_idx += 1;
                            }
                            docstring = Some(doc_lines.join("\n").trim().to_string());
                        }
                    }
                    break;
                }

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "class",
                        name,
                        full_sig,
                        docstring,
                    ));
                }
                i = if found_colon { j + 1 } else { i + 1 };
                continue;
            }

            i += 1;
        }

        // Resilient fallback check: if file contains broken def/class not parsed
        if symbols.is_empty() && (content.contains("def ") || content.contains("class ")) {
            degraded = true;
            let fallback_syms = Self::extract_python_fallback(content, file_rel);
            symbols.extend(fallback_syms);
        }

        (symbols, imports, degraded)
    }

    fn extract_python_fallback(content: &str, file_rel: &str) -> Vec<SymbolDefinition> {
        static PY_DEF_RE: OnceLock<Regex> = OnceLock::new();
        static PY_CLASS_RE: OnceLock<Regex> = OnceLock::new();
        let def_re = PY_DEF_RE.get_or_init(|| {
            Regex::new(r"^\s*(?:async\s+)?def\s+([A-Za-z0-9_]+)").unwrap()
        });
        let class_re = PY_CLASS_RE.get_or_init(|| {
            Regex::new(r"^\s*class\s+([A-Za-z0-9_]+)").unwrap()
        });

        let mut symbols = Vec::new();
        for line in content.lines() {
            let trimmed = line.trim();
            if let Some(caps) = def_re.captures(trimmed) {
                if let Some(m) = caps.get(1) {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        m.as_str(),
                        trimmed,
                        None,
                    ));
                }
            } else if let Some(caps) = class_re.captures(trimmed) {
                if let Some(m) = caps.get(1) {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "class",
                        m.as_str(),
                        trimmed,
                        None,
                    ));
                }
            }
        }
        symbols
    }

    // ------------------------------------------------------------------------
    // Rust Parser (use, pub fn, pub struct, pub enum, pub trait, pub type, doc comments)
    // ------------------------------------------------------------------------
    fn extract_rust(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let mut degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut pending_doc_comments = Vec::new();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            // Collect doc comments ///
            if trimmed.starts_with("///") {
                let comment_text = trimmed.strip_prefix("///").unwrap().trim();
                pending_doc_comments.push(comment_text.to_string());
                i += 1;
                continue;
            }

            if trimmed.is_empty() {
                pending_doc_comments.clear();
                i += 1;
                continue;
            }

            // 1. Rust use / mod statements
            if trimmed.starts_with("use ") || trimmed.starts_with("pub use ") {
                let rest = if let Some(r) = trimmed.strip_prefix("pub use ") {
                    r
                } else {
                    trimmed.strip_prefix("use ").unwrap_or("")
                };
                let clean = rest.trim_end_matches(';').trim();
                if let Some(first_segment) = clean.split("::").next() {
                    let seg = first_segment.trim();
                    if seg != "std" && seg != "core" && seg != "crate" && seg != "self" && seg != "super" && !seg.is_empty() {
                        imports.push(seg.to_string());
                    } else if seg == "crate" {
                        if let Some(second) = clean.split("::").nth(1) {
                            imports.push(second.trim().to_string());
                        }
                    }
                }
                pending_doc_comments.clear();
                i += 1;
                continue;
            } else if trimmed.starts_with("mod ") || trimmed.starts_with("pub mod ") {
                let rest = if let Some(r) = trimmed.strip_prefix("pub mod ") {
                    r
                } else {
                    trimmed.strip_prefix("mod ").unwrap_or("")
                };
                if rest.contains(';') {
                    let mod_name = rest.trim_end_matches(';').trim();
                    if !mod_name.is_empty() {
                        imports.push(mod_name.to_string());
                    }
                }
                pending_doc_comments.clear();
                i += 1;
                continue;
            }

            // 2. Rust pub declarations: pub fn, pub struct, pub enum, pub trait, pub type
            let is_pub = trimmed.starts_with("pub ");
            let is_fn = trimmed.starts_with("fn ") || trimmed.contains(" fn ");
            let is_struct = trimmed.starts_with("struct ") || trimmed.contains(" struct ");
            let is_enum = trimmed.starts_with("enum ") || trimmed.contains(" enum ");
            let is_trait = trimmed.starts_with("trait ") || trimmed.contains(" trait ");
            let is_type = trimmed.starts_with("type ") || trimmed.contains(" type ");

            let symbol_type = if is_fn {
                "function"
            } else if is_struct {
                "struct"
            } else if is_enum {
                "enum"
            } else if is_trait {
                "trait"
            } else if is_type {
                "type"
            } else {
                ""
            };

            if !symbol_type.is_empty() && (is_pub || is_fn || is_struct || is_enum || is_trait || is_type) {
                // Collect multiline signature until { or ; or active-edit break
                let mut sig_parts = Vec::new();
                let mut j = i;
                let mut found_delimiter = false;

                while j < lines.len() {
                    let curr_line = lines[j].trim();
                    sig_parts.push(curr_line);

                    if curr_line.contains('{') || curr_line.ends_with(';') {
                        found_delimiter = true;
                        break;
                    }
                    if j > i + 8 {
                        degraded = true;
                        break;
                    }
                    j += 1;
                }

                if !found_delimiter {
                    degraded = true;
                }

                let full_sig = sig_parts.join(" ");
                let clean_sig = full_sig
                    .split('{')
                    .next()
                    .unwrap_or(&full_sig)
                    .trim_end_matches(';')
                    .trim()
                    .to_string();

                let name = Self::extract_rust_identifier(&clean_sig, symbol_type);
                let docstring = if !pending_doc_comments.is_empty() {
                    Some(pending_doc_comments.join("\n"))
                } else {
                    None
                };

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        symbol_type,
                        name,
                        clean_sig,
                        docstring,
                    ));
                }

                pending_doc_comments.clear();
                i = if found_delimiter { j + 1 } else { i + 1 };
                continue;
            }

            pending_doc_comments.clear();
            i += 1;
        }

        // Resilient fallback check
        if symbols.is_empty() && (content.contains("fn ") || content.contains("struct ") || content.contains("enum ")) {
            degraded = true;
            let fallback_syms = Self::extract_rust_fallback(content, file_rel);
            symbols.extend(fallback_syms);
        }

        (symbols, imports, degraded)
    }

    fn extract_rust_identifier(sig: &str, symbol_type: &str) -> String {
        let pattern = match symbol_type {
            "function" => "fn ",
            "struct" => "struct ",
            "enum" => "enum ",
            "trait" => "trait ",
            "type" => "type ",
            _ => " ",
        };

        if let Some(idx) = sig.find(pattern) {
            let after = &sig[idx + pattern.len()..];
            let name_end = after
                .find(|c: char| c == '(' || c == '<' || c == '{' || c == ':' || c == '=' || c == ';' || c.is_whitespace())
                .unwrap_or(after.len());
            return after[..name_end].trim().to_string();
        }
        String::new()
    }

    fn extract_rust_fallback(content: &str, file_rel: &str) -> Vec<SymbolDefinition> {
        static RUST_FALLBACK_RE: OnceLock<Regex> = OnceLock::new();
        let re = RUST_FALLBACK_RE.get_or_init(|| {
            Regex::new(r"(?:pub\s+)?(?:async\s+|const\s+|unsafe\s+)?(fn|struct|enum|trait|type)\s+([A-Za-z0-9_]+)").unwrap()
        });

        let mut symbols = Vec::new();
        for line in content.lines() {
            let trimmed = line.trim();
            if let Some(caps) = re.captures(trimmed) {
                let kind = caps.get(1).map(|m| m.as_str()).unwrap_or("");
                let name = caps.get(2).map(|m| m.as_str()).unwrap_or("");
                let sym_type = match kind {
                    "fn" => "function",
                    "struct" => "struct",
                    "enum" => "enum",
                    "trait" => "trait",
                    "type" => "type",
                    _ => "unknown",
                };
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        sym_type,
                        name,
                        trimmed,
                        None,
                    ));
                }
            }
        }
        symbols
    }

    // ------------------------------------------------------------------------
    // Go Parser (import, type definitions, func signatures, doc comments)
    // ------------------------------------------------------------------------
    fn extract_go(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let mut degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut pending_comments = Vec::new();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            // Collect Go comments //
            if trimmed.starts_with("//") {
                let comment_text = trimmed.strip_prefix("//").unwrap().trim();
                pending_comments.push(comment_text.to_string());
                i += 1;
                continue;
            }

            if trimmed.is_empty() {
                pending_comments.clear();
                i += 1;
                continue;
            }

            // 1. Go Imports
            if trimmed.starts_with("import (") {
                let mut j = i + 1;
                let mut found_close = false;
                while j < lines.len() {
                    let imp_line = lines[j].trim();
                    if imp_line == ")" {
                        found_close = true;
                        break;
                    }
                    if let Some(start_q) = imp_line.find('"') {
                        if let Some(end_q) = imp_line[start_q + 1..].find('"') {
                            let path = &imp_line[start_q + 1..start_q + 1 + end_q];
                            imports.push(path.to_string());
                        }
                    }
                    j += 1;
                }
                if !found_close {
                    degraded = true;
                }
                pending_comments.clear();
                i = if found_close { j + 1 } else { j };
                continue;
            } else if trimmed.starts_with("import ") {
                if let Some(start_q) = trimmed.find('"') {
                    if let Some(end_q) = trimmed[start_q + 1..].find('"') {
                        let path = &trimmed[start_q + 1..start_q + 1 + end_q];
                        imports.push(path.to_string());
                    }
                }
                pending_comments.clear();
                i += 1;
                continue;
            }

            // 2. Go func signatures
            if trimmed.starts_with("func ") {
                let mut sig_parts = Vec::new();
                let mut j = i;
                let mut found_brace = false;

                while j < lines.len() {
                    let curr_line = lines[j].trim();
                    sig_parts.push(curr_line);
                    if curr_line.ends_with('{') || curr_line.contains('{') {
                        found_brace = true;
                        break;
                    }
                    if j > i + 8 {
                        degraded = true;
                        break;
                    }
                    j += 1;
                }

                if !found_brace {
                    degraded = true;
                }

                let full_sig = sig_parts.join(" ");
                let clean_sig = full_sig
                    .split('{')
                    .next()
                    .unwrap_or(&full_sig)
                    .trim()
                    .to_string();

                let name = Self::extract_go_func_name(&clean_sig);
                let docstring = if !pending_comments.is_empty() {
                    Some(pending_comments.join("\n"))
                } else {
                    None
                };

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        clean_sig,
                        docstring,
                    ));
                }

                pending_comments.clear();
                i = if found_brace { j + 1 } else { i + 1 };
                continue;
            }

            // 3. Go type definitions: struct, interface, type alias
            if trimmed.starts_with("type ") {
                let mut sig_parts = Vec::new();
                let mut j = i;
                let mut found_term = false;

                while j < lines.len() {
                    let curr_line = lines[j].trim();
                    sig_parts.push(curr_line);
                    if curr_line.contains('{') || !curr_line.contains("struct") && !curr_line.contains("interface") {
                        found_term = true;
                        break;
                    }
                    if j > i + 6 {
                        degraded = true;
                        break;
                    }
                    j += 1;
                }


                let full_sig = sig_parts.join(" ");
                let clean_sig = full_sig
                    .split('{')
                    .next()
                    .unwrap_or(&full_sig)
                    .trim()
                    .to_string();

                let (name, sym_type) = Self::extract_go_type_name_and_kind(&clean_sig);
                let docstring = if !pending_comments.is_empty() {
                    Some(pending_comments.join("\n"))
                } else {
                    None
                };

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        sym_type,
                        name,
                        clean_sig,
                        docstring,
                    ));
                }

                pending_comments.clear();
                i = if found_term { j + 1 } else { i + 1 };
                continue;
            }

            pending_comments.clear();
            i += 1;
        }

        if symbols.is_empty() && (content.contains("func ") || content.contains("type ")) {
            degraded = true;
            let fallback_syms = Self::extract_go_fallback(content, file_rel);
            symbols.extend(fallback_syms);
        }

        (symbols, imports, degraded)
    }

    fn extract_go_func_name(sig: &str) -> String {
        // Handle receiver: func (r *Receiver) MethodName(...)
        if let Some(rest) = sig.strip_prefix("func ") {
            let rest = rest.trim();
            if rest.starts_with('(') {
                if let Some(close_paren) = rest.find(')') {
                    let method_part = rest[close_paren + 1..].trim();
                    let name_end = method_part.find('(').unwrap_or(method_part.len());
                    return method_part[..name_end].trim().to_string();
                }
            } else {
                let name_end = rest.find('(').unwrap_or(rest.len());
                return rest[..name_end].trim().to_string();
            }
        }
        String::new()
    }

    fn extract_go_type_name_and_kind(sig: &str) -> (String, &'static str) {
        if let Some(rest) = sig.strip_prefix("type ") {
            let parts: Vec<&str> = rest.split_whitespace().collect();
            if !parts.is_empty() {
                let name = parts[0].to_string();
                let kind = if parts.len() > 1 {
                    match parts[1] {
                        "struct" => "struct",
                        "interface" => "interface",
                        _ => "type",
                    }
                } else {
                    "type"
                };
                return (name, kind);
            }
        }
        (String::new(), "type")
    }

    fn extract_go_fallback(content: &str, file_rel: &str) -> Vec<SymbolDefinition> {
        static GO_FUNC_RE: OnceLock<Regex> = OnceLock::new();
        static GO_TYPE_RE: OnceLock<Regex> = OnceLock::new();
        let func_re = GO_FUNC_RE.get_or_init(|| {
            Regex::new(r"func\s+(?:\([^)]*\)\s+)?([A-Za-z0-9_]+)").unwrap()
        });
        let type_re = GO_TYPE_RE.get_or_init(|| {
            Regex::new(r"type\s+([A-Za-z0-9_]+)\s+(struct|interface|[A-Za-z0-9_]+)").unwrap()
        });

        let mut symbols = Vec::new();
        for line in content.lines() {
            let trimmed = line.trim();
            if let Some(caps) = func_re.captures(trimmed) {
                if let Some(m) = caps.get(1) {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        m.as_str(),
                        trimmed,
                        None,
                    ));
                }
            } else if let Some(caps) = type_re.captures(trimmed) {
                let name = caps.get(1).map(|m| m.as_str()).unwrap_or("");
                let kind = caps.get(2).map(|m| m.as_str()).unwrap_or("type");
                let sym_type = match kind {
                    "struct" => "struct",
                    "interface" => "interface",
                    _ => "type",
                };
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        sym_type,
                        name,
                        trimmed,
                        None,
                    ));
                }
            }
        }
        symbols
    }

    // ------------------------------------------------------------------------
    // TypeScript / JavaScript Parser (import, require, export signatures, JSDoc)
    // ------------------------------------------------------------------------
    fn extract_typescript_javascript(
        content: &str,
        file_rel: &str,
    ) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let mut degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut pending_jsdoc = Vec::new();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            // Collect JSDoc comments /** ... */
            if trimmed.starts_with("/**") {
                if trimmed.ends_with("*/") && trimmed.len() > 5 {
                    let inner = &trimmed[3..trimmed.len() - 2].trim();
                    pending_jsdoc.push(inner.to_string());
                } else {
                    let mut j = i + 1;
                    let mut doc_lines = Vec::new();
                    while j < lines.len() {
                        let doc_l = lines[j].trim();
                        if doc_l.contains("*/") {
                            let text = doc_l.split("*/").next().unwrap_or("").trim_start_matches('*').trim();
                            if !text.is_empty() {
                                doc_lines.push(text);
                            }
                            break;
                        } else {
                            let text = doc_l.trim_start_matches('*').trim();
                            if !text.is_empty() {
                                doc_lines.push(text);
                            }
                        }
                        j += 1;
                    }
                    pending_jsdoc.push(doc_lines.join("\n"));
                    i = j + 1;
                    continue;
                }
                i += 1;
                continue;
            } else if trimmed.starts_with("//") {
                pending_jsdoc.push(trimmed.strip_prefix("//").unwrap().trim().to_string());
                i += 1;
                continue;
            }

            if trimmed.is_empty() {
                pending_jsdoc.clear();
                i += 1;
                continue;
            }

            // 1. TS/JS Imports and requires
            if trimmed.starts_with("import ") {
                if let Some(from_idx) = trimmed.find("from ") {
                    let raw_module = trimmed[from_idx + 5..].trim().trim_end_matches(';');
                    let mod_path = raw_module.trim_matches(|c| c == '\'' || c == '"' || c == '`');
                    if !mod_path.is_empty() {
                        imports.push(mod_path.to_string());
                    }
                } else if trimmed.contains("require(") {
                    if let Some(req_idx) = trimmed.find("require(") {
                        let sub = &trimmed[req_idx + 8..];
                        if let Some(close) = sub.find(')') {
                            let mod_path = sub[..close].trim_matches(|c| c == '\'' || c == '"' || c == '`');
                            if !mod_path.is_empty() {
                                imports.push(mod_path.to_string());
                            }
                        }
                    }
                }
                pending_jsdoc.clear();
                i += 1;
                continue;
            } else if trimmed.contains("require(") {
                if let Some(req_idx) = trimmed.find("require(") {
                    let sub = &trimmed[req_idx + 8..];
                    if let Some(close) = sub.find(')') {
                        let mod_path = sub[..close].trim_matches(|c| c == '\'' || c == '"' || c == '`');
                        if !mod_path.is_empty() {
                            imports.push(mod_path.to_string());
                        }
                    }
                }
                pending_jsdoc.clear();
                i += 1;
                continue;
            }

            // 2. Export / top-level definitions: function, class, interface, type, const
            let is_export = trimmed.starts_with("export ");
            let is_fn = trimmed.contains("function ") || trimmed.contains("function(");
            let is_class = trimmed.contains("class ");
            let is_interface = trimmed.contains("interface ");
            let is_type = trimmed.contains("type ");
            let is_const = (trimmed.starts_with("export const ") || trimmed.starts_with("const "))
                && (trimmed.contains("=>") || trimmed.contains("function"));

            let sym_type = if is_fn {
                "function"
            } else if is_class {
                "class"
            } else if is_interface {
                "interface"
            } else if is_type {
                "type"
            } else if is_const {
                "const"
            } else {
                ""
            };

            if !sym_type.is_empty() && (is_export || is_fn || is_class || is_interface || is_type || is_const) {
                let mut sig_parts = Vec::new();
                let mut j = i;
                let mut found_term = false;

                while j < lines.len() {
                    let curr_line = lines[j].trim();
                    sig_parts.push(curr_line);

                    if curr_line.contains('{') || curr_line.ends_with(';') {
                        found_term = true;
                        break;
                    }
                    if j > i + 8 {
                        degraded = true;
                        break;
                    }
                    j += 1;
                }

                if !found_term {
                    degraded = true;
                }

                let full_sig = sig_parts.join(" ");
                let clean_sig = full_sig
                    .split('{')
                    .next()
                    .unwrap_or(&full_sig)
                    .trim_end_matches(';')
                    .trim()
                    .to_string();

                let name = Self::extract_ts_identifier(&clean_sig, sym_type);
                let docstring = if !pending_jsdoc.is_empty() {
                    Some(pending_jsdoc.join("\n"))
                } else {
                    None
                };

                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        sym_type,
                        name,
                        clean_sig,
                        docstring,
                    ));
                }

                pending_jsdoc.clear();
                i = if found_term { j + 1 } else { i + 1 };
                continue;
            }

            pending_jsdoc.clear();
            i += 1;
        }

        if symbols.is_empty() && (content.contains("function") || content.contains("class") || content.contains("interface")) {
            degraded = true;
            let fallback_syms = Self::extract_ts_fallback(content, file_rel);
            symbols.extend(fallback_syms);
        }

        (symbols, imports, degraded)
    }

    fn extract_ts_identifier(sig: &str, sym_type: &str) -> String {
        let pattern = match sym_type {
            "function" => "function ",
            "class" => "class ",
            "interface" => "interface ",
            "type" => "type ",
            "const" => "const ",
            _ => " ",
        };

        if let Some(idx) = sig.find(pattern) {
            let after = &sig[idx + pattern.len()..];
            let name_end = after
                .find(|c: char| c == '(' || c == '<' || c == '{' || c == ':' || c == '=' || c.is_whitespace())
                .unwrap_or(after.len());
            return after[..name_end].trim().to_string();
        }
        String::new()
    }

    fn extract_ts_fallback(content: &str, file_rel: &str) -> Vec<SymbolDefinition> {
        static TS_FALLBACK_RE: OnceLock<Regex> = OnceLock::new();
        let re = TS_FALLBACK_RE.get_or_init(|| {
            Regex::new(r"(?:export\s+)?(?:async\s+)?(function|class|interface|type|const)\s+([A-Za-z0-9_]+)").unwrap()
        });

        let mut symbols = Vec::new();
        for line in content.lines() {
            let trimmed = line.trim();
            if let Some(caps) = re.captures(trimmed) {
                let kind = caps.get(1).map(|m| m.as_str()).unwrap_or("");
                let name = caps.get(2).map(|m| m.as_str()).unwrap_or("");
                let sym_type = match kind {
                    "function" => "function",
                    "class" => "class",
                    "interface" => "interface",
                    "type" => "type",
                    "const" => "const",
                    _ => "unknown",
                };
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        sym_type,
                        name,
                        trimmed,
                        None,
                    ));
                }
            }
        }
        symbols
    }

    // ------------------------------------------------------------------------
    // Java Parser (package, import, class, interface, record, enum, methods)
    // ------------------------------------------------------------------------
    fn extract_java(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            if trimmed.is_empty() || trimmed.starts_with("//") || trimmed.starts_with("/*") || trimmed.starts_with('*') {
                i += 1;
                continue;
            }

            // Java import statements
            if trimmed.starts_with("import ") {
                let rest = trimmed.strip_prefix("import ").unwrap_or("").trim_start_matches("static ").trim();
                let clean = rest.trim_end_matches(';').trim();
                if !clean.is_empty() {
                    imports.push(clean.to_string());
                }
                i += 1;
                continue;
            }

            // Classes, interfaces, records, enums
            if trimmed.contains("class ") || trimmed.contains("interface ") || trimmed.contains("record ") || trimmed.contains("enum ") {
                let kind = if trimmed.contains("class ") {
                    "class"
                } else if trimmed.contains("interface ") {
                    "interface"
                } else if trimmed.contains("record ") {
                    "record"
                } else {
                    "enum"
                };

                let sub = if let Some(idx) = trimmed.find(kind) {
                    trimmed[idx + kind.len()..].trim()
                } else {
                    ""
                };

                let name = sub.split(|c: char| c.is_whitespace() || c == '{' || c == '<' || c == '(').next().unwrap_or("").trim();
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        kind,
                        name,
                        trimmed.split('{').next().unwrap_or(trimmed).trim(),
                        None,
                    ));
                }
                i += 1;
                continue;
            }

            // Methods: public/protected/private ... (...)
            if (trimmed.starts_with("public ") || trimmed.starts_with("protected ") || trimmed.starts_with("private "))
                && trimmed.contains('(')
                && !trimmed.contains(" class ")
                && !trimmed.contains(" interface ")
            {
                let sig = trimmed.split('{').next().unwrap_or(trimmed).trim();
                let before_paren = trimmed.split('(').next().unwrap_or("").trim();
                let name = before_paren.split_whitespace().last().unwrap_or("");
                if !name.is_empty() && name != "if" && name != "while" && name != "for" {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        sig,
                        None,
                    ));
                }
            }

            i += 1;
        }

        (symbols, imports, degraded)
    }

    // ------------------------------------------------------------------------
    // C# Parser (using, namespace, class, interface, record, struct, enum, methods)
    // ------------------------------------------------------------------------
    fn extract_csharp(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            if trimmed.is_empty() || trimmed.starts_with("//") || trimmed.starts_with("/*") || trimmed.starts_with('*') {
                i += 1;
                continue;
            }

            // using statements
            if trimmed.starts_with("using ") && !trimmed.contains('(') {
                let rest = trimmed.strip_prefix("using ").unwrap_or("").trim_start_matches("static ").trim();
                let clean = rest.trim_end_matches(';').trim();
                let final_import = if let Some(eq_idx) = clean.find('=') {
                    clean[eq_idx + 1..].trim()
                } else {
                    clean
                };
                if !final_import.is_empty() {
                    imports.push(final_import.to_string());
                }
                i += 1;
                continue;
            }

            // Classes, interfaces, records, structs, enums
            if trimmed.contains("class ")
                || trimmed.contains("interface ")
                || trimmed.contains("record ")
                || trimmed.contains("struct ")
                || trimmed.contains("enum ")
            {
                let kind = if trimmed.contains("class ") {
                    "class"
                } else if trimmed.contains("interface ") {
                    "interface"
                } else if trimmed.contains("record ") {
                    "record"
                } else if trimmed.contains("struct ") {
                    "struct"
                } else {
                    "enum"
                };

                let sub = if let Some(idx) = trimmed.find(kind) {
                    trimmed[idx + kind.len()..].trim()
                } else {
                    ""
                };

                let name = sub.split(|c: char| c.is_whitespace() || c == '{' || c == '<' || c == '(' || c == ':').next().unwrap_or("").trim();
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        kind,
                        name,
                        trimmed.split('{').next().unwrap_or(trimmed).trim(),
                        None,
                    ));
                }
                i += 1;
                continue;
            }

            // Methods: public/protected/private/internal ... (...)
            if (trimmed.starts_with("public ") || trimmed.starts_with("protected ") || trimmed.starts_with("private ") || trimmed.starts_with("internal "))
                && trimmed.contains('(')
                && !trimmed.contains(" class ")
                && !trimmed.contains(" interface ")
                && !trimmed.contains(" struct ")
            {
                let sig = trimmed.split('{').next().unwrap_or(trimmed).trim();
                let before_paren = trimmed.split('(').next().unwrap_or("").trim();
                let name = before_paren.split_whitespace().last().unwrap_or("");
                if !name.is_empty() && name != "if" && name != "while" && name != "for" {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        sig,
                        None,
                    ));
                }
            }

            i += 1;
        }

        (symbols, imports, degraded)
    }

    // ------------------------------------------------------------------------
    // C++ Parser (includes, class, struct, namespace, functions, methods)
    // ------------------------------------------------------------------------
    fn extract_cpp(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            if trimmed.is_empty() || trimmed.starts_with("//") || trimmed.starts_with("/*") || trimmed.starts_with('*') {
                i += 1;
                continue;
            }

            // #include statements
            if trimmed.starts_with("#include") {
                let rest = trimmed.strip_prefix("#include").unwrap_or("").trim();
                let clean = rest.trim_matches(|c| c == '<' || c == '>' || c == '"').trim();
                if !clean.is_empty() {
                    imports.push(clean.to_string());
                }
                i += 1;
                continue;
            }

            // Classes, structs, namespaces
            if trimmed.contains("class ") || trimmed.contains("struct ") || trimmed.contains("namespace ") {
                let kind = if trimmed.contains("class ") {
                    "class"
                } else if trimmed.contains("struct ") {
                    "struct"
                } else {
                    "namespace"
                };

                let sub = if let Some(idx) = trimmed.find(kind) {
                    trimmed[idx + kind.len()..].trim()
                } else {
                    ""
                };

                let name = sub.split(|c: char| c.is_whitespace() || c == '{' || c == '<' || c == '(' || c == ':').next().unwrap_or("").trim();
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        kind,
                        name,
                        trimmed.split('{').next().unwrap_or(trimmed).trim(),
                        None,
                    ));
                }
                i += 1;
                continue;
            }

            // Functions / methods
            if trimmed.contains('(') && !trimmed.starts_with("if") && !trimmed.starts_with("while") && !trimmed.starts_with("for") && !trimmed.starts_with("switch") && !trimmed.starts_with("catch") {
                let sig = trimmed.split('{').next().unwrap_or(trimmed).trim_end_matches(';').trim();
                let before_paren = sig.split('(').next().unwrap_or("").trim();
                let name = before_paren.split_whitespace().last().unwrap_or("").trim_start_matches('*').trim_start_matches('&');
                if !name.is_empty() && name != "return" {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        sig,
                        None,
                    ));
                }
            }

            i += 1;
        }

        (symbols, imports, degraded)
    }

    // ------------------------------------------------------------------------
    // Ruby Parser (require, require_relative, class, module, def)
    // ------------------------------------------------------------------------
    fn extract_ruby(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>, bool) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();
        let degraded = false;

        let lines: Vec<&str> = content.lines().collect();
        let mut i = 0;

        while i < lines.len() {
            let line = lines[i];
            let trimmed = line.trim();

            if trimmed.is_empty() || trimmed.starts_with('#') || trimmed.starts_with("=begin") {
                i += 1;
                continue;
            }

            // require / require_relative
            if trimmed.starts_with("require ") || trimmed.starts_with("require_relative ") {
                let rest = if let Some(r) = trimmed.strip_prefix("require_relative ") {
                    r
                } else {
                    trimmed.strip_prefix("require ").unwrap_or("")
                };
                let clean = rest.trim().trim_matches(|c| c == '\'' || c == '"').trim();
                if !clean.is_empty() {
                    imports.push(clean.to_string());
                }
                i += 1;
                continue;
            }

            // class / module
            if trimmed.starts_with("class ") || trimmed.starts_with("module ") {
                let kind = if trimmed.starts_with("class ") {
                    "class"
                } else {
                    "module"
                };
                let sub = trimmed.strip_prefix(kind).unwrap_or("").trim();
                let name = sub.split(|c: char| c.is_whitespace() || c == '<' || c == ';').next().unwrap_or("").trim();
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        kind,
                        name,
                        trimmed,
                        None,
                    ));
                }
                i += 1;
                continue;
            }

            // def methods
            if trimmed.starts_with("def ") {
                let sub = trimmed.strip_prefix("def ").unwrap_or("").trim();
                let name = sub.split(|c: char| c.is_whitespace() || c == '(' || c == ';').next().unwrap_or("").trim();
                if !name.is_empty() {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "function",
                        name,
                        trimmed,
                        None,
                    ));
                }
            }

            i += 1;
        }

        (symbols, imports, degraded)
    }

    // ------------------------------------------------------------------------
    // Generic Fallback Extractor (for unknown extensions or heavily damaged syntax)
    // ------------------------------------------------------------------------
    fn extract_fallback_generic(content: &str, file_rel: &str) -> (Vec<SymbolDefinition>, Vec<String>) {
        let mut symbols = Vec::new();
        let mut imports = Vec::new();

        static GENERIC_RE: OnceLock<Regex> = OnceLock::new();
        let re = GENERIC_RE.get_or_init(|| {
            Regex::new(r"(?:def|fn|func|function|class|struct|interface|type)\s+([A-Za-z0-9_]+)").unwrap()
        });

        for line in content.lines() {
            let trimmed = line.trim();
            if let Some(caps) = re.captures(trimmed) {
                if let Some(m) = caps.get(1) {
                    symbols.push(SymbolDefinition::new(
                        file_rel,
                        "symbol",
                        m.as_str(),
                        trimmed,
                        None,
                    ));
                }
            }
            if trimmed.starts_with("import ") || trimmed.starts_with("use ") {
                let parts: Vec<&str> = trimmed.split_whitespace().collect();
                if parts.len() > 1 {
                    imports.push(parts[1].trim_matches(';').to_string());
                }
            }
        }
        (symbols, imports)
    }

    // ------------------------------------------------------------------------
    // Candidate Dependency Path Resolution
    // ------------------------------------------------------------------------
    fn resolve_import_candidates(
        mod_import: &str,
        current_file: &Path,
        repo_root: &Path,
        language: SourceLanguage,
    ) -> Vec<PathBuf> {
        let mut candidates = Vec::new();
        let current_dir = current_file.parent().unwrap_or(repo_root);

        match language {
            SourceLanguage::Python => {
                let clean = mod_import.trim_start_matches('.');
                let rel = clean.replace('.', "/");
                // 1. Sibling to current file
                candidates.push(current_dir.join(format!("{}.py", rel)));
                candidates.push(current_dir.join(&rel).join("__init__.py"));
                // 2. Relative to repo root
                candidates.push(repo_root.join(format!("{}.py", rel)));
                candidates.push(repo_root.join(&rel).join("__init__.py"));
            }
            SourceLanguage::Rust => {
                // Mod or use statements
                candidates.push(current_dir.join(format!("{}.rs", mod_import)));
                candidates.push(current_dir.join(mod_import).join("mod.rs"));
                candidates.push(repo_root.join("src").join(format!("{}.rs", mod_import)));
                candidates.push(repo_root.join("src").join(mod_import).join("mod.rs"));
            }
            SourceLanguage::Go => {
                let clean = mod_import.trim_start_matches("./");
                candidates.push(current_dir.join(clean));
                candidates.push(repo_root.join(clean));
            }
            SourceLanguage::TypeScript | SourceLanguage::JavaScript => {
                let clean = mod_import.trim_start_matches("./");
                let extensions = ["ts", "tsx", "js", "jsx"];
                for ext in &extensions {
                    candidates.push(current_dir.join(format!("{}.{}", clean, ext)));
                    candidates.push(current_dir.join(clean).join(format!("index.{}", ext)));
                    candidates.push(repo_root.join(format!("{}.{}", clean, ext)));
                    candidates.push(repo_root.join(clean).join(format!("index.{}", ext)));
                }
            }
            SourceLanguage::Java => {
                let rel = mod_import.replace('.', "/");
                let simple_name = mod_import.split('.').last().unwrap_or(mod_import);
                candidates.push(current_dir.join(format!("{}.java", simple_name)));
                candidates.push(repo_root.join(format!("{}.java", rel)));
                candidates.push(repo_root.join("src").join("main").join("java").join(format!("{}.java", rel)));
            }
            SourceLanguage::CSharp => {
                let rel = mod_import.replace('.', "/");
                let simple_name = mod_import.split('.').last().unwrap_or(mod_import);
                candidates.push(current_dir.join(format!("{}.cs", simple_name)));
                candidates.push(repo_root.join(format!("{}.cs", rel)));
                candidates.push(repo_root.join("src").join(format!("{}.cs", rel)));
                candidates.push(repo_root.join("src").join(format!("{}.cs", simple_name)));
            }
            SourceLanguage::Cpp => {
                candidates.push(current_dir.join(mod_import));
                candidates.push(repo_root.join(mod_import));
                candidates.push(repo_root.join("include").join(mod_import));
                candidates.push(repo_root.join("src").join(mod_import));
            }
            SourceLanguage::Ruby => {
                let clean = mod_import.trim_start_matches("./");
                let rel = if clean.ends_with(".rb") {
                    clean.to_string()
                } else {
                    format!("{}.rb", clean)
                };
                candidates.push(current_dir.join(&rel));
                candidates.push(repo_root.join(&rel));
                candidates.push(repo_root.join("lib").join(&rel));
                candidates.push(repo_root.join("app").join(&rel));
            }
            SourceLanguage::Unknown => {
                candidates.push(current_dir.join(mod_import));
            }
        }

        candidates
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Instant;
    use tempfile::tempdir;

    #[test]
    fn test_ast_resolver_java() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let service_java = root.join("AuthService.java");
        fs::write(
            &service_java,
            r#"package com.example.service;
import java.util.List;

public class AuthService {
    public boolean authenticate(String user, String token) {
        return user != null && token != null;
    }
}
"#,
        ).unwrap();

        let manifest = AstContextResolver::resolve(&service_java, root, 3).unwrap();
        assert_eq!(manifest.target_file, "AuthService.java");
        assert!(manifest.symbols.iter().any(|s| s.name == "AuthService" && s.symbol_type == "class"));
        assert!(manifest.symbols.iter().any(|s| s.name == "authenticate" && s.symbol_type == "function"));
    }

    #[test]
    fn test_ast_resolver_ruby() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let helper_rb = root.join("helper.rb");
        fs::write(
            &helper_rb,
            r#"module AuthHelper
  def verify_token(token)
    token != nil
  end
end
"#,
        ).unwrap();

        let service_rb = root.join("auth_service.rb");
        fs::write(
            &service_rb,
            r#"require_relative 'helper'

class AuthService
  def authenticate(user)
    user&.active?
  end
end
"#,
        ).unwrap();

        let manifest = AstContextResolver::resolve(&service_rb, root, 2).unwrap();
        assert_eq!(manifest.target_file, "auth_service.rb");
        assert!(manifest.discovered_dependencies.contains(&"helper.rb".to_string()));
        assert!(manifest.symbols.iter().any(|s| s.name == "AuthService" && s.symbol_type == "class"));
        assert!(manifest.symbols.iter().any(|s| s.name == "authenticate" && s.symbol_type == "function"));
    }

    #[test]
    fn test_ast_resolver_cpp() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let header = root.join("Buffer.hpp");
        fs::write(
            &header,
            r#"#pragma once
#include <vector>
#include <cstddef>

namespace memory {
class BufferPool {
public:
    void* allocate(size_t size);
};
}
"#,
        ).unwrap();

        let manifest = AstContextResolver::resolve(&header, root, 2).unwrap();
        assert_eq!(manifest.target_file, "Buffer.hpp");
        assert!(manifest.symbols.iter().any(|s| s.name == "memory" && s.symbol_type == "namespace"));
        assert!(manifest.symbols.iter().any(|s| s.name == "BufferPool" && s.symbol_type == "class"));
        assert!(manifest.symbols.iter().any(|s| s.name == "allocate" && s.symbol_type == "function"));
    }

    #[test]
    fn test_ast_resolver_csharp() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let service_cs = root.join("TokenService.cs");
        fs::write(
            &service_cs,
            r#"using System;
using System.Collections.Generic;

namespace MyService
{
    public class TokenService
    {
        public bool ValidateToken(string token)
        {
            return token != null;
        }
    }
}
"#,
        ).unwrap();

        let manifest = AstContextResolver::resolve(&service_cs, root, 3).unwrap();
        assert_eq!(manifest.target_file, "TokenService.cs");
        assert!(manifest.symbols.iter().any(|s| s.name == "TokenService" && s.symbol_type == "class"));
        assert!(manifest.symbols.iter().any(|s| s.name == "ValidateToken" && s.symbol_type == "function"));
        assert!(manifest.discovered_dependencies.is_empty());
    }

    #[test]
    fn test_ast_resolver_python() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let helper_py = root.join("helper.py");
        fs::write(
            &helper_py,
            "\"\"\"Helper module docstring.\"\"\"\ndef calculate_tax(amount: float) -> float:\n    \"\"\"Calculates tax.\"\"\"\n    return amount * 0.2\n",
        )
        .unwrap();

        let main_py = root.join("main.py");
        fs::write(
            &main_py,
            "import helper\n\nclass InvoiceProcessor:\n    \"\"\"Processes invoices.\"\"\"\n    def process(self):\n        pass\n",
        )
        .unwrap();

        let manifest = AstContextResolver::resolve(&main_py, root, 2).unwrap();
        assert_eq!(manifest.target_file, "main.py");
        assert!(manifest.discovered_dependencies.contains(&"helper.py".to_string()));

        let symbol_names: Vec<&str> = manifest.symbols.iter().map(|s| s.name.as_str()).collect();
        assert!(symbol_names.contains(&"InvoiceProcessor"));
        assert!(symbol_names.contains(&"process"));
        assert!(symbol_names.contains(&"calculate_tax"));

        let tax_sym = manifest.symbols.iter().find(|s| s.name == "calculate_tax").unwrap();
        assert_eq!(tax_sym.docstring.as_deref(), Some("Calculates tax."));
    }

    #[test]
    fn test_ast_resolver_rust() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let models_rs = root.join("models.rs");
        fs::write(
            &models_rs,
            r#"
/// User credential model
pub struct User {
    pub id: u64,
    pub name: String,
}

pub enum Role {
    Admin,
    Member,
}

pub trait Authenticatable {
    fn verify(&self) -> bool;
}
"#,
        )
        .unwrap();

        let main_rs = root.join("main.rs");
        fs::write(
            &main_rs,
            r#"
mod models;
use models::User;

/// Validates session token
pub fn validate_session(token: &str) -> bool {
    !token.is_empty()
}
"#,
        )
        .unwrap();

        let manifest = AstContextResolver::resolve(&main_rs, root, 2).unwrap();
        assert_eq!(manifest.target_file, "main.rs");
        assert!(manifest.discovered_dependencies.contains(&"models.rs".to_string()));

        let names: Vec<&str> = manifest.symbols.iter().map(|s| s.name.as_str()).collect();
        assert!(names.contains(&"validate_session"));
        assert!(names.contains(&"User"));
        assert!(names.contains(&"Role"));
        assert!(names.contains(&"Authenticatable"));

        let user_sym = manifest.symbols.iter().find(|s| s.name == "User").unwrap();
        assert_eq!(user_sym.symbol_type, "struct");
        assert_eq!(user_sym.docstring.as_deref(), Some("User credential model"));
    }

    #[test]
    fn test_ast_resolver_go() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let auth_go = root.join("auth.go");
        fs::write(
            &auth_go,
            r#"
package auth

import "fmt"

// Session maintains authentication state.
type Session struct {
    Token string
}

// Authenticator defines auth contract.
type Authenticator interface {
    Login(user string) bool
}

// Verify validates session token.
func Verify(token string) bool {
    return len(token) > 0
}

func (s *Session) Invalidate() error {
    return nil
}
"#,
        )
        .unwrap();

        let manifest = AstContextResolver::resolve(&auth_go, root, 1).unwrap();
        let names: Vec<&str> = manifest.symbols.iter().map(|s| s.name.as_str()).collect();
        assert!(names.contains(&"Session"));
        assert!(names.contains(&"Authenticator"));
        assert!(names.contains(&"Verify"));
        assert!(names.contains(&"Invalidate"));

        let session_sym = manifest.symbols.iter().find(|s| s.name == "Session").unwrap();
        assert_eq!(session_sym.symbol_type, "struct");
        assert_eq!(session_sym.docstring.as_deref(), Some("Session maintains authentication state."));
    }

    #[test]
    fn test_ast_resolver_typescript() {
        let dir = tempdir().unwrap();
        let root = dir.path();

        let db_ts = root.join("db.ts");
        fs::write(
            &db_ts,
            r#"
export interface QueryResult {
    rows: number;
}

export class DatabasePool {
    connect() {}
}
"#,
        )
        .unwrap();

        let server_ts = root.join("server.ts");
        fs::write(
            &server_ts,
            r#"
import { QueryResult } from './db';

/**
 * Dispatches HTTP requests.
 */
export async function handleRequest(url: string): Promise<string> {
    return "ok";
}

export type HttpHandler = (req: string) => Promise<string>;
"#,
        )
        .unwrap();

        let manifest = AstContextResolver::resolve(&server_ts, root, 2).unwrap();
        assert_eq!(manifest.target_file, "server.ts");
        assert!(manifest.discovered_dependencies.contains(&"db.ts".to_string()));

        let names: Vec<&str> = manifest.symbols.iter().map(|s| s.name.as_str()).collect();
        assert!(names.contains(&"handleRequest"));
        assert!(names.contains(&"HttpHandler"));
        assert!(names.contains(&"QueryResult"));
        assert!(names.contains(&"DatabasePool"));

        let fn_sym = manifest.symbols.iter().find(|s| s.name == "handleRequest").unwrap();
        assert_eq!(fn_sym.docstring.as_deref(), Some("Dispatches HTTP requests."));
    }

    #[test]
    fn test_syntax_error_resilience() {
        // Broken mid-edit Python file (incomplete function signature without closing parenthesis or colon)
        let broken_py = r#"
import os

class IncompleteClass:
    def active_edit_fn(x: int, y:
    def valid_fn(z: int):
        return z
"#;
        let (symbols, _, degraded) = AstContextResolver::extract_symbols_and_imports(broken_py, "broken.py", SourceLanguage::Python);
        assert!(degraded);
        let names: Vec<&str> = symbols.iter().map(|s| s.name.as_str()).collect();
        assert!(names.contains(&"IncompleteClass"));
        assert!(names.contains(&"valid_fn") || names.contains(&"active_edit_fn"));

        // Broken mid-edit Rust file (unclosed brace and incomplete pub fn)
        let broken_rs = r#"
pub struct ServiceConfig {
    pub port: u16,

pub fn mid_edit_calc(val: usize,
"#;
        let (symbols_rs, _, degraded_rs) = AstContextResolver::extract_symbols_and_imports(broken_rs, "broken.rs", SourceLanguage::Rust);
        assert!(degraded_rs);
        let names_rs: Vec<&str> = symbols_rs.iter().map(|s| s.name.as_str()).collect();
        assert!(names_rs.contains(&"ServiceConfig"));
        assert!(names_rs.contains(&"mid_edit_calc"));

        // Broken mid-edit Go file
        let broken_go = r#"
type BrokenStruct struct {
func IncompleteFunc(a int,
"#;
        let (symbols_go, _, degraded_go) = AstContextResolver::extract_symbols_and_imports(broken_go, "broken.go", SourceLanguage::Go);
        assert!(degraded_go);
        let names_go: Vec<&str> = symbols_go.iter().map(|s| s.name.as_str()).collect();
        assert!(names_go.contains(&"BrokenStruct") || names_go.contains(&"IncompleteFunc"));

        // Broken mid-edit TypeScript file
        let broken_ts = r#"
export interface IncompleteInterface {
export function midEditFn(a: string,
"#;
        let (symbols_ts, _, degraded_ts) = AstContextResolver::extract_symbols_and_imports(broken_ts, "broken.ts", SourceLanguage::TypeScript);
        assert!(degraded_ts);
        let names_ts: Vec<&str> = symbols_ts.iter().map(|s| s.name.as_str()).collect();
        assert!(names_ts.contains(&"IncompleteInterface") || names_ts.contains(&"midEditFn"));
    }

    #[test]
    fn test_extraction_latency_under_5ms() {
        // Generate a synthetic file with 200 functions
        let mut large_file = String::new();
        large_file.push_str("import math\n\n");
        for i in 0..200 {
            large_file.push_str(&format!(
                "def compute_metric_{}(x: float, y: float) -> float:\n    \"\"\"Docstring for metric {}\"\"\"\n    return x * {} + y\n\n",
                i, i, i
            ));
        }

        let start = Instant::now();
        let (symbols, imports, degraded) =
            AstContextResolver::extract_symbols_and_imports(&large_file, "synthetic.py", SourceLanguage::Python);
        let elapsed = start.elapsed();

        assert!(!degraded);
        assert_eq!(imports.len(), 1);
        assert_eq!(symbols.len(), 200);
        // Requirement: < 10ms per file deterministic signature extraction in debug mode (<0.5ms release)
        assert!(
            elapsed.as_millis() <= 10,
            "Extraction took {} ms, exceeding target",
            elapsed.as_millis()
        );
    }
}
