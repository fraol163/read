use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs::{self, File, OpenOptions};
use std::io::{BufRead, BufReader, Write};
#[cfg(unix)]
use std::os::unix::io::AsRawFd;
use std::path::{Path, PathBuf};

use crate::hasher::DiffHasher;

pub const GENESIS_HASH: &str = "0000000000000000000000000000000000000000000000000000000000000000";
pub const MAX_WAL_ENTRIES: usize = 10_000;
pub const MAX_WAL_BYTES: u64 = 5 * 1024 * 1024; // 5 MB

#[derive(Debug, Serialize, Deserialize, PartialEq, Clone)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Idle,
    Defined,
    Executing,
    Verified,
    HandoffComplete,
    Blocked,
}

#[derive(Debug, Serialize, Deserialize, Clone, PartialEq)]
pub struct CasLock {
    pub expected_revision: u64,
    pub locked_by: String,
    pub lease_expires: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Clone, PartialEq)]
pub struct LearnProgress {
    pub current_scenario: usize,
    pub completed_scenarios: Vec<usize>,
    pub attempts: std::collections::HashMap<String, usize>,
    pub mastery_score: f64,
}

#[derive(Debug, Serialize, Deserialize, Clone, PartialEq)]
pub struct State {
    pub state_revision: u64,
    pub status: TaskStatus,
    pub active_task_id: Option<String>,
    pub contract_hash: Option<String>,
    pub base_commit: Option<String>,
    pub staged_diff_hash: Option<String>,
    pub verify_hash: Option<String>,
    pub cas_lock: CasLock,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub learn_progress: Option<LearnProgress>,
}

impl State {
    pub fn new_idle() -> Self {
        Self {
            state_revision: 0,
            status: TaskStatus::Idle,
            active_task_id: None,
            contract_hash: None,
            base_commit: None,
            staged_diff_hash: None,
            verify_hash: None,
            cas_lock: CasLock {
                expected_revision: 0,
                locked_by: "system".to_string(),
                lease_expires: Utc::now() + chrono::Duration::hours(24),
            },
            learn_progress: None,
        }
    }
}

pub struct FileLock {
    file: File,
}

impl FileLock {
    pub fn acquire_exclusive<P: AsRef<Path>>(path: P) -> Result<Self, String> {
        let p = path.as_ref();
        if let Some(parent) = p.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| format!("Failed to create parent dir for lock: {}", e))?;
        }

        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(false)
            .open(p)
            .map_err(|e| format!("Failed to open lock file: {}", e))?;

        #[cfg(unix)]
        {
            let fd = file.as_raw_fd();
            let ret = unsafe { libc::flock(fd, libc::LOCK_EX) };
            if ret != 0 {
                return Err("Failed to acquire exclusive POSIX lock".to_string());
            }
        }

        Ok(Self { file })
    }
}

impl Drop for FileLock {
    fn drop(&mut self) {
        #[cfg(unix)]
        {
            let fd = self.file.as_raw_fd();
            unsafe {
                libc::flock(fd, libc::LOCK_UN);
            }
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct WalEntry {
    pub timestamp: String,
    pub prev_hash: String,
    pub event_type: String,
    pub payload: String,
    pub entry_hash: String,
}

impl WalEntry {
    pub fn format_line(&self) -> String {
        format!(
            "{} | prev_hash={} | {} | {} | entry_hash={}\n",
            self.timestamp, self.prev_hash, self.event_type, self.payload, self.entry_hash
        )
    }

    pub fn compute_hash(timestamp: &str, prev_hash: &str, event_type: &str, payload: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(timestamp.as_bytes());
        hasher.update(b"|");
        hasher.update(prev_hash.as_bytes());
        hasher.update(b"|");
        hasher.update(event_type.as_bytes());
        hasher.update(b"|");
        hasher.update(payload.as_bytes());
        format!("{:x}", hasher.finalize())
    }

    pub fn parse(line: &str) -> Result<Self, String> {
        let trimmed = line.trim_end_matches(['\r', '\n']);
        if trimmed.is_empty() {
            return Err("Empty WAL line".to_string());
        }

        let (left, entry_hash_part) = trimmed
            .rsplit_once(" | entry_hash=")
            .ok_or_else(|| format!("Malformed WAL line: missing entry_hash in '{}'", trimmed))?;

        let entry_hash = entry_hash_part.trim().to_string();
        if entry_hash.len() != 64 {
            return Err(format!("Invalid entry_hash length in WAL line: '{}'", entry_hash));
        }

        let parts: Vec<&str> = left.splitn(4, " | ").collect();
        if parts.len() < 4 {
            return Err(format!("Malformed WAL line: insufficient fields in '{}'", trimmed));
        }

        let timestamp = parts[0].trim().to_string();
        let prev_part = parts[1].trim();
        let prev_hash = prev_part
            .strip_prefix("prev_hash=")
            .ok_or_else(|| format!("Malformed WAL line: missing 'prev_hash=' prefix in '{}'", prev_part))?
            .to_string();

        if prev_hash.len() != 64 {
            return Err(format!("Invalid prev_hash length in WAL line: '{}'", prev_hash));
        }

        let event_type = parts[2].trim().to_string();
        let payload = parts[3].to_string();

        let expected_hash = Self::compute_hash(&timestamp, &prev_hash, &event_type, &payload);
        if expected_hash != entry_hash {
            return Err(format!(
                "WAL entry hash mismatch! Computed '{}' but found '{}'",
                expected_hash, entry_hash
            ));
        }

        Ok(WalEntry {
            timestamp,
            prev_hash,
            event_type,
            payload,
            entry_hash,
        })
    }
}

pub struct StateEngine {
    root: PathBuf,
    state_dir: PathBuf,
    state_file: PathBuf,
    wal_file: PathBuf,
    lock_file: PathBuf,
}

impl StateEngine {
    pub fn new<P: AsRef<Path>>(root: P) -> Self {
        let root = root.as_ref().to_path_buf();
        let read_dir = root.join(".read");
        let legacy_dir = root.join(".method");
        if legacy_dir.exists() && !read_dir.exists() {
            let _ = fs::rename(&legacy_dir, &read_dir);
        }
        let state_dir = read_dir;
        let state_file = state_dir.join("STATE.json");
        let wal_file = state_dir.join("WAL.log");
        let lock_file = state_dir.join(".state.lock");
        Self {
            root,
            state_dir,
            state_file,
            wal_file,
            lock_file,
        }
    }

    /// Initializes .read directory and initial idle STATE.json with cryptographic genesis WAL.
    pub fn init(&self) -> Result<(), String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;

        fs::create_dir_all(&self.state_dir)
            .map_err(|e| format!("Failed to create .read dir: {}", e))?;

        if self.state_file.exists() {
            return Err("READ framework already initialized in this directory".to_string());
        }

        let initial_state = State::new_idle();
        self.atomic_save(&initial_state)?;
        self.internal_append_wal("INIT", "Initialized READ framework state engine")?;
        Ok(())
    }

    /// Loads current state from STATE.json.
    pub fn load(&self) -> Result<State, String> {
        if !self.state_file.exists() {
            return Err("STATE.json does not exist. Run read init first.".to_string());
        }
        let content = fs::read_to_string(&self.state_file)
            .map_err(|e| format!("Failed to read STATE.json: {}", e))?;
        serde_json::from_str(&content)
            .map_err(|e| format!("Corrupt STATE.json: {}", e))
    }

    /// Atomic write to STATE.json using POSIX .tmp file swap + fsync.
    pub fn atomic_save(&self, state: &State) -> Result<(), String> {
        let pid = std::process::id();
        let tmp_file = self.state_dir.join(format!("STATE.json.tmp.{}", pid));
        let data = serde_json::to_string_pretty(state)
            .map_err(|e| format!("Failed to serialize state: {}", e))?;

        {
            let mut file = File::create(&tmp_file)
                .map_err(|e| format!("Failed to create temp state file: {}", e))?;
            file.write_all(data.as_bytes())
                .map_err(|e| format!("Failed to write temp state file: {}", e))?;
            file.flush()
                .map_err(|e| format!("Failed to flush: {}", e))?;
            file.sync_all()
                .map_err(|e| format!("Failed to fsync: {}", e))?;
        }

        fs::rename(&tmp_file, &self.state_file)
            .map_err(|e| format!("Atomic POSIX rename failed: {}", e))?;

        Ok(())
    }

    /// Validates the full cryptographic SHA-256 chain in .method/WAL.log.
    pub fn verify_wal(&self) -> Result<Vec<WalEntry>, String> {
        if !self.wal_file.exists() {
            return Err("WAL.log does not exist".to_string());
        }

        let content = fs::read_to_string(&self.wal_file)
            .map_err(|e| format!("Failed to read WAL.log: {}", e))?;

        let lines: Vec<&str> = content.lines().filter(|l| !l.trim().is_empty()).collect();
        if lines.is_empty() {
            return Ok(Vec::new());
        }

        let mut entries = Vec::with_capacity(lines.len());
        let mut expected_prev_hash: Option<String> = None;

        for (idx, line) in lines.iter().enumerate() {
            let entry = WalEntry::parse(line)
                .map_err(|e| format!("WAL corruption detected at line {}: {}", idx + 1, e))?;

            if let Some(ref prev) = expected_prev_hash {
                if entry.prev_hash != *prev {
                    return Err(format!(
                        "WAL chain broken at line {}: expected prev_hash='{}', found '{}'",
                        idx + 1,
                        prev,
                        entry.prev_hash
                    ));
                }
            } else if entry.event_type != "CHECKPOINT" && entry.prev_hash != GENESIS_HASH {
                return Err(format!(
                    "WAL chain genesis error at line 1: expected prev_hash='{}', found '{}'",
                    GENESIS_HASH, entry.prev_hash
                ));
            }

            expected_prev_hash = Some(entry.entry_hash.clone());
            entries.push(entry);
        }

        Ok(entries)
    }

    /// Appends an event to WAL with lock acquisition.
    pub fn append_wal(&self, event_type: &str, payload: &str) -> Result<WalEntry, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        self.internal_append_wal(event_type, payload)
    }

    /// Internal append assuming lock already held.
    fn internal_append_wal(&self, event_type: &str, payload: &str) -> Result<WalEntry, String> {
        fs::create_dir_all(&self.state_dir)
            .map_err(|e| format!("Failed to create .read dir: {}", e))?;

        let prev_hash = if self.wal_file.exists() {
            let entries = self.verify_wal()?;
            if let Some(last) = entries.last() {
                last.entry_hash.clone()
            } else {
                GENESIS_HASH.to_string()
            }
        } else {
            GENESIS_HASH.to_string()
        };

        let timestamp = Utc::now().to_rfc3339();
        let entry_hash = WalEntry::compute_hash(&timestamp, &prev_hash, event_type, payload);
        let entry = WalEntry {
            timestamp,
            prev_hash,
            event_type: event_type.to_string(),
            payload: payload.to_string(),
            entry_hash,
        };

        let log_line = entry.format_line();
        {
            let mut file = OpenOptions::new()
                .create(true)
                .append(true)
                .open(&self.wal_file)
                .map_err(|e| format!("Failed to open WAL.log: {}", e))?;
            file.write_all(log_line.as_bytes())
                .map_err(|e| format!("Failed to append to WAL.log: {}", e))?;
            file.flush()
                .map_err(|e| format!("Failed to flush WAL: {}", e))?;
            file.sync_all()
                .map_err(|e| format!("Failed to fsync WAL: {}", e))?;
        }

        self.maybe_compact_wal()?;

        Ok(entry)
    }

    /// Checks if WAL exceeds 10,000 entries or 5 MB, and triggers compaction.
    pub fn maybe_compact_wal(&self) -> Result<(), String> {
        if !self.wal_file.exists() {
            return Ok(());
        }

        let metadata = fs::metadata(&self.wal_file)
            .map_err(|e| format!("Failed to inspect WAL metadata: {}", e))?;

        let mut should_compact = metadata.len() >= MAX_WAL_BYTES;
        if !should_compact {
            let file = File::open(&self.wal_file)
                .map_err(|e| format!("Failed to open WAL: {}", e))?;
            let reader = BufReader::new(file);
            let count = reader.lines().count();
            if count >= MAX_WAL_ENTRIES {
                should_compact = true;
            }
        }

        if should_compact {
            self.compact_wal()?;
        }

        Ok(())
    }

    /// Safely compacts WAL into a consolidated checkpoint snapshot.
    pub fn compact_wal(&self) -> Result<(), String> {
        let entries = self.verify_wal()?;
        if entries.is_empty() {
            return Ok(());
        }

        let last_entry_hash = entries.last().unwrap().entry_hash.clone();
        let st = self.load()?;

        let snapshot = serde_json::json!({
            "snapshot_revision": st.state_revision,
            "status": st.status,
            "active_task_id": st.active_task_id,
            "contract_hash": st.contract_hash,
            "base_commit": st.base_commit,
            "staged_diff_hash": st.staged_diff_hash,
            "verify_hash": st.verify_hash,
            "cas_lock": st.cas_lock,
        });
        let payload = serde_json::to_string(&snapshot)
            .map_err(|e| format!("Failed to serialize snapshot: {}", e))?;

        let timestamp = Utc::now().to_rfc3339();
        let entry_hash = WalEntry::compute_hash(&timestamp, &last_entry_hash, "CHECKPOINT", &payload);
        let checkpoint_entry = WalEntry {
            timestamp,
            prev_hash: last_entry_hash,
            event_type: "CHECKPOINT".to_string(),
            payload,
            entry_hash,
        };

        let tmp_wal = self.state_dir.join("WAL.log.tmp");
        {
            let mut file = File::create(&tmp_wal)
                .map_err(|e| format!("Failed to create WAL.log.tmp: {}", e))?;
            file.write_all(checkpoint_entry.format_line().as_bytes())
                .map_err(|e| format!("Failed to write WAL.log.tmp: {}", e))?;
            file.flush()
                .map_err(|e| format!("Failed to flush WAL tmp: {}", e))?;
            file.sync_all()
                .map_err(|e| format!("Failed to fsync WAL tmp: {}", e))?;
        }

        fs::rename(&tmp_wal, &self.wal_file)
            .map_err(|e| format!("Failed to atomically rename compacted WAL: {}", e))?;

        Ok(())
    }

    /// Verifies lease validity and expected revision monotonicity.
    fn check_lease_and_revision(
        &self,
        st: &State,
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<(), String> {
        if let Some(exp) = expected_revision {
            if st.state_revision != exp {
                return Err(format!(
                    "CasLockViolation: CAS revision mismatch / revision collision: expected revision {} but current revision is {}",
                    exp, st.state_revision
                ));
            }
        }

        let now = Utc::now();
        if st.cas_lock.locked_by != "system"
            && st.cas_lock.locked_by != caller
            && now < st.cas_lock.lease_expires
        {
            return Err(format!(
                "CasLockViolation: CAS lock violation: active lease held by '{}' until {}",
                st.cas_lock.locked_by,
                st.cas_lock.lease_expires.to_rfc3339()
            ));
        }

        Ok(())
    }

    /// Explicitly acquires a CAS lease for a worker.
    pub fn acquire_lease(
        &self,
        worker_id: &str,
        duration_secs: i64,
        expected_revision: Option<u64>,
    ) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, worker_id)?;

        st.state_revision += 1;
        st.cas_lock.expected_revision = st.state_revision;
        st.cas_lock.locked_by = worker_id.to_string();
        st.cas_lock.lease_expires = Utc::now() + chrono::Duration::seconds(duration_secs);

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "LEASE_ACQUIRE",
            &format!(
                "locked_by={} lease_expires={} revision={}",
                worker_id,
                st.cas_lock.lease_expires.to_rfc3339(),
                st.state_revision
            ),
        )?;
        Ok(st)
    }

    /// Releases a CAS lease back to system idle.
    pub fn release_lease(&self, worker_id: &str) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;

        if st.cas_lock.locked_by != worker_id && st.cas_lock.locked_by != "system" {
            let now = Utc::now();
            if now < st.cas_lock.lease_expires {
                return Err(format!(
                    "Cannot release lease: held by '{}', not caller '{}'",
                    st.cas_lock.locked_by, worker_id
                ));
            }
        }

        st.state_revision += 1;
        st.cas_lock.expected_revision = st.state_revision;
        st.cas_lock.locked_by = "system".to_string();
        st.cas_lock.lease_expires = Utc::now() + chrono::Duration::hours(24);

        self.atomic_save(&st)?;
        self.internal_append_wal("LEASE_RELEASE", &format!("released_by={}", worker_id))?;
        Ok(st)
    }

    /// Transitions to 'defined' stage: freezes contract hash.
    pub fn transition_define(
        &self,
        task_id: &str,
        contract_hash: &str,
        base_commit: &str,
    ) -> Result<State, String> {
        self.transition_define_cas(task_id, contract_hash, base_commit, None, "system")
    }

    pub fn transition_define_cas(
        &self,
        task_id: &str,
        contract_hash: &str,
        base_commit: &str,
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, caller)?;

        if st.status != TaskStatus::Idle && st.status != TaskStatus::HandoffComplete {
            return Err(format!(
                "CasLockViolation: Cannot define task: Active task already in status {:?}",
                st.status
            ));
        }

        st.state_revision += 1;
        st.status = TaskStatus::Defined;
        st.active_task_id = Some(task_id.to_string());
        st.contract_hash = Some(contract_hash.to_string());
        st.base_commit = Some(base_commit.to_string());
        st.staged_diff_hash = None;
        st.verify_hash = None;
        st.cas_lock.expected_revision = st.state_revision;

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "DEFINE",
            &format!("task_id={} contract_hash={} base_commit={}", task_id, contract_hash, base_commit),
        )?;
        Ok(st)
    }

    /// Transitions to 'executing' stage (Build).
    pub fn transition_start_build(&self) -> Result<State, String> {
        self.transition_start_build_cas(None, "system")
    }

    pub fn transition_start_build_cas(
        &self,
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, caller)?;

        if st.status != TaskStatus::Defined {
            return Err(format!("CasLockViolation: Task must be in 'defined' status, currently {:?}", st.status));
        }

        st.state_revision += 1;
        st.status = TaskStatus::Executing;
        st.cas_lock.expected_revision = st.state_revision;

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "BUILD_START",
            &format!("task_id={}", st.active_task_id.as_deref().unwrap_or("none")),
        )?;
        Ok(st)
    }

    /// Records verification pass (AROP) and seals verify_hash.
    pub fn transition_record_verification(
        &self,
        staged_diff_hash: &str,
        verify_output_bytes: &[u8],
    ) -> Result<State, String> {
        self.transition_record_verification_cas(staged_diff_hash, verify_output_bytes, None, "system")
    }

    pub fn transition_record_verification_cas(
        &self,
        staged_diff_hash: &str,
        verify_output_bytes: &[u8],
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, caller)?;

        if st.status != TaskStatus::Executing {
            return Err(format!("CasLockViolation: Task must be in 'executing' status, currently {:?}", st.status));
        }

        let mut hasher = Sha256::new();
        hasher.update(verify_output_bytes);
        let verify_hash = format!("{:x}", hasher.finalize());

        st.state_revision += 1;
        st.status = TaskStatus::Verified;
        st.staged_diff_hash = Some(staged_diff_hash.to_string());
        st.verify_hash = Some(verify_hash.clone());
        st.cas_lock.expected_revision = st.state_revision;

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "VERIFIED",
            &format!("staged_diff_hash={} verify_hash={}", staged_diff_hash, verify_hash),
        )?;
        Ok(st)
    }

    /// Mechanically enforced CAS transition to 'handoff_complete'.
    pub fn transition_handoff(&self) -> Result<State, String> {
        let current_diff = DiffHasher::hash_staged_diff(&self.root).ok().map(|(h, _)| h);
        self.transition_handoff_with_diff(current_diff.as_deref())
    }

    /// Handoff with explicit diff comparison for strict drift enforcement.
    pub fn transition_handoff_with_diff(&self, current_diff_hash: Option<&str>) -> Result<State, String> {
        self.transition_handoff_cas(current_diff_hash, None, "system")
    }

    pub fn transition_handoff_cas(
        &self,
        current_diff_hash: Option<&str>,
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<State, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, caller)?;

        if st.status != TaskStatus::Verified {
            return Err(format!(
                "CasLockViolation: Cannot transition to handoff. Status must be 'verified', currently {:?}",
                st.status
            ));
        }

        let verify_hash = match &st.verify_hash {
            Some(vh) if !vh.trim().is_empty() => vh.clone(),
            _ => {
                return Err("CasLockViolation: verify_hash is null. Verification gate was bypassed!".to_string());
            }
        };

        let entries = self.verify_wal()?;
        if verify_hash.starts_with("OVR-") {
            let authorized = entries.iter().any(|e| {
                e.event_type == "MASTER_OVERRIDE" && e.payload.contains(&format!("override_hash={}", verify_hash))
            });
            if !authorized {
                return Err(format!(
                    "CasLockViolation: Master Override hash '{}' is not registered in WAL audit trail!",
                    verify_hash
                ));
            }
        } else {
            let authorized = entries.iter().any(|e| {
                e.event_type == "VERIFIED" && e.payload.contains(&format!("verify_hash={}", verify_hash))
            });
            if !authorized {
                return Err(format!(
                    "CasLockViolation: State tampering detected! verify_hash '{}' has no matching cryptographic audit record in WAL.log",
                    verify_hash
                ));
            }

            let expected_diff = st.staged_diff_hash.as_ref().ok_or_else(|| {
                "CasLockViolation: staged_diff_hash is missing for verified task".to_string()
            })?;

            if let Some(actual_diff) = current_diff_hash {
                if expected_diff != actual_diff {
                    return Err(format!(
                        "CasLockViolation: Staged diff hash drift detected! Verified diff is '{}' but current diff is '{}'. Re-verification required.",
                        expected_diff, actual_diff
                    ));
                }
            }
        }

        st.state_revision += 1;
        st.status = TaskStatus::HandoffComplete;
        st.cas_lock.expected_revision = st.state_revision;

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "HANDOFF",
            &format!(
                "task_id={} verify_hash={}",
                st.active_task_id.as_deref().unwrap_or("none"),
                st.verify_hash.as_deref().unwrap_or("none")
            ),
        )?;
        let _ = self.anchor_wal_in_git();
        Ok(st)
    }

    /// The Master Override: allows the human operator to bypass a failed gate with mandatory rationale.
    /// Creates a hashed and immutably logged tombstone in WAL.log.
    /// Provides integrity and auditability via WAL chaining; does NOT provide non-repudiation.
    pub fn master_override(
        &self,
        operator_id: &str,
        rationale: &str,
    ) -> Result<State, String> {
        self.master_override_cas(operator_id, rationale, None, "system")
    }

    pub fn master_override_cas(
        &self,
        operator_id: &str,
        rationale: &str,
        expected_revision: Option<u64>,
        caller: &str,
    ) -> Result<State, String> {
        if rationale.trim().len() < 30 {
            return Err("Master Override rejected: Rationale must be at least 30 characters explaining the engineering justification".to_string());
        }

        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        let mut st = self.load()?;
        self.check_lease_and_revision(&st, expected_revision, caller)?;

        let timestamp = Utc::now().to_rfc3339();

        let mut hasher = Sha256::new();
        hasher.update(rationale.as_bytes());
        hasher.update(operator_id.as_bytes());
        hasher.update(timestamp.as_bytes());
        let override_hash = format!("OVR-{:x}", hasher.finalize());

        st.state_revision += 1;
        st.status = TaskStatus::Verified;
        st.verify_hash = Some(override_hash.clone());
        st.cas_lock.expected_revision = st.state_revision;

        self.atomic_save(&st)?;
        self.internal_append_wal(
            "MASTER_OVERRIDE",
            &format!(
                "operator={} override_hash={} rationale=\"{}\"",
                operator_id, override_hash, rationale
            ),
        )?;
        Ok(st)
    }

    /// Anchors the latest WAL head hash into a git tag named 'read-wal-head' if inside a git repository.
    pub fn anchor_wal_in_git(&self) -> Result<String, String> {
        let entries = self.verify_wal()?;
        let last = entries.last().ok_or_else(|| "No WAL entries to anchor".to_string())?;
        let head_hash = &last.entry_hash;
        if self.root.join(".git").exists() {
            // Check if HEAD exists before attempting to tag
            let head_check = std::process::Command::new("git")
                .args(["rev-parse", "--verify", "HEAD"])
                .current_dir(&self.root)
                .output();
            if let Ok(out) = head_check {
                if out.status.success() {
                    let status = std::process::Command::new("git")
                        .args(["tag", "-f", "-a", "read-wal-head", "-m", head_hash])
                        .current_dir(&self.root)
                        .status()
                        .map_err(|e| format!("Failed to execute git tag: {}", e))?;
                    if !status.success() {
                        return Err("git tag read-wal-head command failed".to_string());
                    }
                }
            }
        }
        Ok(head_hash.clone())
    }

    /// Verifies that the git tag 'read-wal-head' matches the latest WAL entry hash.
    pub fn verify_wal_anchor(&self) -> Result<(), String> {
        if self.root.join(".git").exists() {
            let output = std::process::Command::new("git")
                .args(["tag", "-l", "--format=%(contents)", "read-wal-head"])
                .current_dir(&self.root)
                .output()
                .map_err(|e| format!("Failed to inspect git tag: {}", e))?;
            let tag_msg = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !tag_msg.is_empty() {
                let entries = self.verify_wal()?;
                if let Some(last) = entries.last() {
                    if !tag_msg.contains(&last.entry_hash) {
                        return Err(format!(
                            "WAL git anchor mismatch: git tag read-wal-head ({}) does not match WAL head hash ({})",
                            tag_msg, last.entry_hash
                        ));
                    }
                }
            }
        }
        Ok(())
    }

    /// Crash recovery: validates WAL cryptographic integrity and STATE.json.
    /// Resumes without re-spending context tokens!
    pub fn recover(&self, current_staged_diff_hash: Option<&str>) -> Result<String, String> {
        let _lock = FileLock::acquire_exclusive(&self.lock_file)?;
        self.verify_wal_anchor()?;
        let st = self.load()?;

        let entries = self.verify_wal()?;
        if entries.is_empty() {
            return Err("WAL is empty or missing".to_string());
        }

        let last_entry = entries.last().unwrap();
        let last_log = last_entry.format_line();

        // Validate diff consistency if staged
        if let Some(expected_diff) = &st.staged_diff_hash {
            if let Some(actual_diff) = current_staged_diff_hash {
                if expected_diff != actual_diff {
                    return Err(format!(
                        "Recovery Integrity Warning: Staged diff hash drift. Expected {} but found {}",
                        expected_diff, actual_diff
                    ));
                }
            }
        }

        Ok(format!(
            "RECOVER SUCCESS (0 Tokens Re-spent):\n- Status: {:?}\n- Task ID: {}\n- State Revision: {}\n- Last WAL Event: {}",
            st.status,
            st.active_task_id.as_deref().unwrap_or("none"),
            st.state_revision,
            last_log.trim()
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn test_lifecycle_and_cas_locks() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());

        // 1. Init
        engine.init().unwrap();
        let st = engine.load().unwrap();
        assert_eq!(st.status, TaskStatus::Idle);
        assert_eq!(st.state_revision, 0);

        // 2. Define
        let st_def = engine.transition_define("TASK-001", "hash_contract_123", "commit_abc").unwrap();
        assert_eq!(st_def.status, TaskStatus::Defined);
        assert_eq!(st_def.state_revision, 1);

        // 3. Try to skip to handoff -> Must fail with StateLockViolation!
        assert!(engine.transition_handoff().is_err());

        // 4. Start build
        let st_bld = engine.transition_start_build().unwrap();
        assert_eq!(st_bld.status, TaskStatus::Executing);
        assert_eq!(st_bld.state_revision, 2);

        // 5. Verify pass
        let st_ver = engine.transition_record_verification("diff_hash_456", b"all 10 mutants killed").unwrap();
        assert_eq!(st_ver.status, TaskStatus::Verified);
        assert!(st_ver.verify_hash.is_some());
        assert_eq!(st_ver.state_revision, 3);

        // 6. Handoff -> Now succeeds!
        let st_hnd = engine.transition_handoff_with_diff(Some("diff_hash_456")).unwrap();
        assert_eq!(st_hnd.status, TaskStatus::HandoffComplete);
        assert_eq!(st_hnd.state_revision, 4);

        // 7. Recover test
        let rec = engine.recover(Some("diff_hash_456")).unwrap();
        assert!(rec.contains("RECOVER SUCCESS (0 Tokens Re-spent)"));
    }

    #[test]
    fn test_cas_revision_monotonicity() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();

        // Attempt transition with invalid expected revision
        let err = engine.transition_define_cas(
            "TASK-001",
            "hash1",
            "commit1",
            Some(999), // expected revision 999 does not match 0
            "system",
        );
        assert!(err.is_err());
        let err_str = err.unwrap_err();
        assert!(err_str.contains("CAS revision mismatch"), "Actual error was: {}", err_str);

        // Valid expected revision succeeds and advances
        let st = engine.transition_define_cas(
            "TASK-001",
            "hash1",
            "commit1",
            Some(0),
            "system",
        ).unwrap();
        assert_eq!(st.state_revision, 1);
    }

    #[test]
    fn test_cas_lease_expiration_and_preemption() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();

        // Worker A acquires lease for 1 second
        engine.acquire_lease("worker_alpha", 1, None).unwrap();

        // Worker B attempts transition while lease is active -> blocked!
        let err = engine.transition_define_cas("TASK-W2", "h", "c", None, "worker_beta");
        assert!(err.is_err());
        assert!(err.unwrap_err().contains("CAS lock violation"));

        // Worker A can transition with its own lease
        let st = engine.transition_define_cas("TASK-W1", "h", "c", None, "worker_alpha").unwrap();
        assert_eq!(st.status, TaskStatus::Defined);

        // Wait for lease to expire
        std::thread::sleep(std::time::Duration::from_millis(1100));

        // Now Worker B can preempt expired lease and start build!
        let st_bld = engine.transition_start_build_cas(None, "worker_beta").unwrap();
        assert_eq!(st_bld.status, TaskStatus::Executing);
    }

    #[test]
    fn test_handoff_unconditionally_blocked_on_diff_drift() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();
        engine.transition_define("TASK-DRIFT", "c_hash", "b_commit").unwrap();
        engine.transition_start_build().unwrap();
        engine.transition_record_verification("diff_original_111", b"ok").unwrap();

        // Attempt handoff with drifted diff -> must fail!
        let err = engine.transition_handoff_with_diff(Some("diff_modified_999"));
        assert!(err.is_err());
        assert!(err.unwrap_err().contains("Staged diff hash drift detected"));

        // Handoff with matching diff -> succeeds!
        let ok = engine.transition_handoff_with_diff(Some("diff_original_111"));
        assert!(ok.is_ok());
    }

    #[test]
    fn test_wal_cryptographic_chaining_and_corruption_detection() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();
        engine.transition_define("TASK-001", "c_hash", "commit").unwrap();
        engine.transition_start_build().unwrap();

        // Verify valid chain
        let entries = engine.verify_wal().unwrap();
        assert_eq!(entries.len(), 3);
        assert_eq!(entries[0].prev_hash, GENESIS_HASH);
        assert_eq!(entries[1].prev_hash, entries[0].entry_hash);
        assert_eq!(entries[2].prev_hash, entries[1].entry_hash);

        // Tamper with WAL file content
        let wal_path = &engine.wal_file;
        let raw = fs::read_to_string(wal_path).unwrap();
        let tampered = raw.replace("c_hash", "corrupted_hash");
        fs::write(&wal_path, tampered).unwrap();

        // verify_wal and recover must detect corruption and halt
        assert!(engine.verify_wal().is_err());
        let rec_err = engine.recover(None);
        assert!(rec_err.is_err());
        assert!(rec_err.unwrap_err().contains("WAL corruption detected"));
    }

    #[test]
    fn test_wal_compaction_preserves_revision_count() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();

        // Create 20 events
        for i in 1..=10 {
            engine.append_wal("EVENT_TEST", &format!("index={}", i)).unwrap();
        }

        let before_entries = engine.verify_wal().unwrap();
        assert_eq!(before_entries.len(), 11); // 1 INIT + 10 EVENT_TEST

        // Run compaction
        engine.compact_wal().unwrap();

        // Compacted WAL should contain exactly 1 CHECKPOINT entry
        let after_entries = engine.verify_wal().unwrap();
        assert_eq!(after_entries.len(), 1);
        assert_eq!(after_entries[0].event_type, "CHECKPOINT");
        assert_eq!(after_entries[0].prev_hash, before_entries.last().unwrap().entry_hash);

        // Ensure state revision is preserved
        let st = engine.load().unwrap();
        assert_eq!(st.state_revision, 0);

        // Next event chains cleanly from checkpoint
        let next_entry = engine.append_wal("AFTER_COMPACT", "payload").unwrap();
        assert_eq!(next_entry.prev_hash, after_entries[0].entry_hash);

        // Recovery succeeds from compacted WAL
        let rec = engine.recover(None).unwrap();
        assert!(rec.contains("RECOVER SUCCESS"));
    }

    #[test]
    fn test_master_override() {
        let dir = tempdir().unwrap();
        let engine = StateEngine::new(dir.path());
        engine.init().unwrap();
        engine.transition_define("TASK-002", "c_hash", "b_commit").unwrap();
        engine.transition_start_build().unwrap();

        // Operator applies override
        let st_ovr = engine.master_override(
            "architect_renan",
            "Bypassing mutant gate because upstream API fixture is flaky but verified manually",
        ).unwrap();
        assert_eq!(st_ovr.status, TaskStatus::Verified);
        assert!(st_ovr.verify_hash.as_ref().unwrap().starts_with("OVR-"));

        // Handoff succeeds with override
        let st_hnd = engine.transition_handoff().unwrap();
        assert_eq!(st_hnd.status, TaskStatus::HandoffComplete);
    }

    #[test]
    fn test_wal_git_anchor_verification_and_rewritten_chain_detection() {
        let dir = tempdir().unwrap();
        let path = dir.path();

        // Initialize git repo in tempdir
        let _ = std::process::Command::new("git").args(["init"]).current_dir(path).output().unwrap();
        let _ = std::process::Command::new("git").args(["config", "user.email", "test@test.local"]).current_dir(path).output().unwrap();
        let _ = std::process::Command::new("git").args(["config", "user.name", "Test Worker"]).current_dir(path).output().unwrap();
        let _ = std::process::Command::new("git").args(["commit", "--allow-empty", "-m", "initial"]).current_dir(path).output().unwrap();

        let engine = StateEngine::new(path);
        engine.init().unwrap();
        engine.transition_define("TASK-ANCHOR", "contract_hash", "base_commit").unwrap();

        // Anchor WAL in git tag
        let head_hash = engine.anchor_wal_in_git().unwrap();
        assert_eq!(head_hash.len(), 64);

        // Verification passes when anchor matches
        assert!(engine.verify_wal_anchor().is_ok());

        // Now simulate adversarial attack: forge a valid internal WAL chain that differs from git tag anchor
        let wal_file = engine.wal_file.clone();
        let timestamp = "2026-11-23T00:00:00Z";
        let prev_hash = GENESIS_HASH;
        let event_type = "FORGED";
        let payload = "hacked=true";
        let forged_hash = WalEntry::compute_hash(timestamp, prev_hash, event_type, payload);
        let forged_line = format!("{} | prev_hash={} | {} | {} | entry_hash={}\n", timestamp, prev_hash, event_type, payload, forged_hash);
        fs::write(&wal_file, forged_line).unwrap();

        // Internal verify_wal passes because chain is self-consistent
        assert!(engine.verify_wal().is_ok());

        // But git anchor verification must fail because git tag points to original head_hash
        let verify_res = engine.verify_wal_anchor();
        assert!(verify_res.is_err());
        let err_msg = verify_res.unwrap_err();
        assert!(err_msg.contains("WAL git anchor mismatch"));
    }

    #[test]
    fn test_migration_from_legacy_method_path() {
        let dir = tempdir().unwrap();
        let path = dir.path();
        let legacy = path.join(".method");
        fs::create_dir_all(&legacy).unwrap();
        fs::write(legacy.join("STATE.json"), "{\"status\": \"idle\"}").unwrap();
        fs::write(legacy.join("WAL.log"), "genesis\n").unwrap();

        // Calling StateEngine::new must migrate .method to .read
        let engine = StateEngine::new(path);
        assert!(!legacy.exists(), "Legacy .method directory should be migrated away");
        assert!(path.join(".read").exists(), "Canonical .read directory should exist");
        assert!(engine.state_file.exists());
        assert!(engine.wal_file.exists());
    }
}
