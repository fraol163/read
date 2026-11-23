use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

fn default_version() -> String {
    "0.7.0".to_string()
}
fn default_host() -> String {
    "127.0.0.1".to_string()
}
fn default_port() -> u16 {
    8089
}
fn default_workspace_root() -> String {
    ".".to_string()
}
fn default_storage_policy() -> String {
    "both_sync".to_string()
}
fn default_sync_interval() -> u64 {
    30
}
fn default_chunk_size() -> usize {
    1048576
}
fn default_exclude_patterns() -> Vec<String> {
    vec![
        ".git".into(),
        ".method".into(),
        "target".into(),
        "node_modules".into(),
        "__pycache__".into(),
        ".venv".into(),
    ]
}
fn default_state_dir() -> String {
    ".method".to_string()
}
fn default_state_file() -> String {
    ".method/STATE.json".to_string()
}
fn default_wal_file() -> String {
    ".method/WAL.log".to_string()
}
fn default_true() -> bool {
    true
}
fn default_wal_compaction_threshold() -> u64 {
    1000
}
fn default_max_depth() -> usize {
    3
}
fn default_ast_timeout() -> u64 {
    100
}
fn default_supported_languages() -> Vec<String> {
    vec![
        "python".into(),
        "typescript".into(),
        "rust".into(),
        "go".into(),
    ]
}
fn default_max_mutants() -> usize {
    10
}
fn default_mutation_timeout() -> u64 {
    4500
}
fn default_mutation_operators() -> Vec<String> {
    vec![
        "INVERT_EQ".into(),
        "FLIP_BOOL_TRUE".into(),
        "FLIP_BOOL_FALSE".into(),
        "BOUNDARY_SHIFT".into(),
        "RETURN_INVERT".into(),
    ]
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct DaemonConfig {
    #[serde(default = "default_host")]
    pub host: String,
    #[serde(default = "default_port")]
    pub port: u16,
    #[serde(default = "default_workspace_root")]
    pub workspace_root: String,
    #[serde(default)]
    pub engine_bin: String,
    #[serde(default)]
    pub custom_harness: String,
}

impl Default for DaemonConfig {
    fn default() -> Self {
        Self {
            host: default_host(),
            port: default_port(),
            workspace_root: default_workspace_root(),
            engine_bin: String::new(),
            custom_harness: String::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct StorageConfig {
    #[serde(default = "default_storage_policy")]
    pub default_policy: String,
    #[serde(default = "default_sync_interval")]
    pub sync_interval_seconds: u64,
    #[serde(default = "default_chunk_size")]
    pub chunk_size_bytes: usize,
    #[serde(default = "default_exclude_patterns")]
    pub exclude_patterns: Vec<String>,
}

impl Default for StorageConfig {
    fn default() -> Self {
        Self {
            default_policy: default_storage_policy(),
            sync_interval_seconds: default_sync_interval(),
            chunk_size_bytes: default_chunk_size(),
            exclude_patterns: default_exclude_patterns(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct CoreConfig {
    #[serde(default = "default_state_dir")]
    pub state_dir: String,
    #[serde(default = "default_state_file")]
    pub state_file: String,
    #[serde(default = "default_wal_file")]
    pub wal_file: String,
    #[serde(default = "default_true")]
    pub strict_cas: bool,
    #[serde(default = "default_wal_compaction_threshold")]
    pub wal_compaction_threshold: u64,
}

impl Default for CoreConfig {
    fn default() -> Self {
        Self {
            state_dir: default_state_dir(),
            state_file: default_state_file(),
            wal_file: default_wal_file(),
            strict_cas: default_true(),
            wal_compaction_threshold: default_wal_compaction_threshold(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct AstConfig {
    #[serde(default = "default_max_depth")]
    pub max_depth: usize,
    #[serde(default = "default_ast_timeout")]
    pub timeout_ms: u64,
    #[serde(default = "default_supported_languages")]
    pub supported_languages: Vec<String>,
}

impl Default for AstConfig {
    fn default() -> Self {
        Self {
            max_depth: default_max_depth(),
            timeout_ms: default_ast_timeout(),
            supported_languages: default_supported_languages(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct MutationConfig {
    #[serde(default = "default_max_mutants")]
    pub max_mutants: usize,
    #[serde(default = "default_mutation_timeout")]
    pub timeout_ms: u64,
    #[serde(default = "default_mutation_operators")]
    pub operators: Vec<String>,
}

impl Default for MutationConfig {
    fn default() -> Self {
        Self {
            max_mutants: default_max_mutants(),
            timeout_ms: default_mutation_timeout(),
            operators: default_mutation_operators(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ReadConfig {
    #[serde(default = "default_version")]
    pub version: String,
    #[serde(default)]
    pub daemon: DaemonConfig,
    #[serde(default)]
    pub storage: StorageConfig,
    #[serde(default)]
    pub core: CoreConfig,
    #[serde(default)]
    pub ast: AstConfig,
    #[serde(default)]
    pub mutation: MutationConfig,
}

impl Default for ReadConfig {
    fn default() -> Self {
        Self {
            version: default_version(),
            daemon: DaemonConfig::default(),
            storage: StorageConfig::default(),
            core: CoreConfig::default(),
            ast: AstConfig::default(),
            mutation: MutationConfig::default(),
        }
    }
}

impl ReadConfig {
    pub fn global_config_path() -> Option<PathBuf> {
        if let Ok(xdg) = std::env::var("XDG_CONFIG_HOME") {
            if !xdg.is_empty() {
                return Some(PathBuf::from(xdg).join("read").join("config.json"));
            }
        }
        if let Ok(home) = std::env::var("HOME") {
            if !home.is_empty() {
                return Some(
                    PathBuf::from(home)
                        .join(".config")
                        .join("read")
                        .join("config.json"),
                );
            }
        }
        None
    }

    pub fn local_config_path(workspace: &Path) -> PathBuf {
        workspace.join(".read").join("config.json")
    }

    pub fn merge_from_json(&mut self, json_val: &serde_json::Value) {
        if let Some(v) = json_val.get("version").and_then(|v| v.as_str()) {
            self.version = v.to_string();
        }

        if let Some(daemon) = json_val.get("daemon") {
            if let Some(h) = daemon.get("host").and_then(|v| v.as_str()) {
                self.daemon.host = h.to_string();
            }
            if let Some(p) = daemon.get("port").and_then(|v| v.as_u64()) {
                self.daemon.port = p as u16;
            }
            if let Some(w) = daemon.get("workspace_root").and_then(|v| v.as_str()) {
                self.daemon.workspace_root = w.to_string();
            }
            if let Some(e) = daemon.get("engine_bin").and_then(|v| v.as_str()) {
                self.daemon.engine_bin = e.to_string();
            }
            if let Some(c) = daemon.get("custom_harness").and_then(|v| v.as_str()) {
                self.daemon.custom_harness = c.to_string();
            }
        }

        if let Some(storage) = json_val.get("storage") {
            if let Some(p) = storage.get("default_policy").and_then(|v| v.as_str()) {
                self.storage.default_policy = p.to_string();
            }
            if let Some(s) = storage
                .get("sync_interval_seconds")
                .and_then(|v| v.as_u64())
            {
                self.storage.sync_interval_seconds = s;
            }
            if let Some(c) = storage.get("chunk_size_bytes").and_then(|v| v.as_u64()) {
                self.storage.chunk_size_bytes = c as usize;
            }
            if let Some(arr) = storage
                .get("exclude_patterns")
                .and_then(|v| v.as_array())
            {
                self.storage.exclude_patterns = arr
                    .iter()
                    .filter_map(|x| x.as_str().map(|s| s.to_string()))
                    .collect();
            }
        }

        if let Some(core) = json_val.get("core") {
            if let Some(d) = core.get("state_dir").and_then(|v| v.as_str()) {
                self.core.state_dir = d.to_string();
            }
            if let Some(f) = core.get("state_file").and_then(|v| v.as_str()) {
                self.core.state_file = f.to_string();
            }
            if let Some(w) = core.get("wal_file").and_then(|v| v.as_str()) {
                self.core.wal_file = w.to_string();
            }
            if let Some(sc) = core.get("strict_cas").and_then(|v| v.as_bool()) {
                self.core.strict_cas = sc;
            }
            if let Some(wt) = core
                .get("wal_compaction_threshold")
                .and_then(|v| v.as_u64())
            {
                self.core.wal_compaction_threshold = wt;
            }
        }

        if let Some(ast) = json_val.get("ast") {
            if let Some(d) = ast.get("max_depth").and_then(|v| v.as_u64()) {
                self.ast.max_depth = d as usize;
            }
            if let Some(t) = ast.get("timeout_ms").and_then(|v| v.as_u64()) {
                self.ast.timeout_ms = t;
            }
            if let Some(arr) = ast.get("supported_languages").and_then(|v| v.as_array()) {
                self.ast.supported_languages = arr
                    .iter()
                    .filter_map(|x| x.as_str().map(|s| s.to_string()))
                    .collect();
            }
        }

        if let Some(mutation) = json_val.get("mutation") {
            if let Some(m) = mutation.get("max_mutants").and_then(|v| v.as_u64()) {
                self.mutation.max_mutants = m as usize;
            }
            if let Some(t) = mutation.get("timeout_ms").and_then(|v| v.as_u64()) {
                self.mutation.timeout_ms = t;
            }
            if let Some(arr) = mutation.get("operators").and_then(|v| v.as_array()) {
                self.mutation.operators = arr
                    .iter()
                    .filter_map(|x| x.as_str().map(|s| s.to_string()))
                    .collect();
            }
        }
    }

    pub fn load(workspace: &Path) -> Self {
        let mut config = Self::default();

        // 1. System config (/etc/read/config.json)
        let sys_path = Path::new("/etc/read/config.json");
        if sys_path.is_file() {
            if let Ok(content) = fs::read_to_string(sys_path) {
                if let Ok(json_val) = serde_json::from_str::<serde_json::Value>(&content) {
                    config.merge_from_json(&json_val);
                }
            }
        }

        // 2. User global config (~/.config/read/config.json)
        if let Some(global_path) = Self::global_config_path() {
            if global_path.is_file() {
                if let Ok(content) = fs::read_to_string(&global_path) {
                    if let Ok(json_val) = serde_json::from_str::<serde_json::Value>(&content) {
                        config.merge_from_json(&json_val);
                    }
                }
            }
        }

        // 3. Project-local config (.read/config.json)
        let local_path = Self::local_config_path(workspace);
        if local_path.is_file() {
            if let Ok(content) = fs::read_to_string(&local_path) {
                if let Ok(json_val) = serde_json::from_str::<serde_json::Value>(&content) {
                    config.merge_from_json(&json_val);
                }
            }
        }

        config
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_default_config() {
        let cfg = ReadConfig::default();
        assert_eq!(cfg.version, "0.7.0");
        assert_eq!(cfg.daemon.port, 8089);
        assert_eq!(cfg.core.strict_cas, true);
        assert_eq!(cfg.mutation.max_mutants, 10);
        assert_eq!(cfg.ast.max_depth, 3);
    }

    #[test]
    fn test_workspace_local_override() {
        let dir = tempdir().unwrap();
        let ws_path = dir.path();
        let dot_read = ws_path.join(".read");
        fs::create_dir_all(&dot_read).unwrap();

        let override_json = r#"{
            "daemon": { "port": 9099 },
            "mutation": { "max_mutants": 5 }
        }"#;
        fs::write(dot_read.join("config.json"), override_json).unwrap();

        let loaded = ReadConfig::load(ws_path);
        assert_eq!(loaded.daemon.port, 9099);
        assert_eq!(loaded.mutation.max_mutants, 5);
        // Default preserved for untouched fields
        assert_eq!(loaded.core.strict_cas, true);
        assert_eq!(loaded.ast.max_depth, 3);
    }
}
