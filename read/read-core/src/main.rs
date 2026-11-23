use read_core::ast::AstContextResolver;
use read_core::config::ReadConfig;
use read_core::hasher::DiffHasher;
use read_core::mutation::BoundedMutator;
use read_core::state::StateEngine;
use std::env;
use std::path::Path;

fn print_usage() {
    println!(
        r#"READ Core Computational Engine (v0.7.0)
Resource-conscious, Evidence-driven AI Development

USAGE:
    read-engine <COMMAND> [ARGS]

COMMANDS:
    init                             Initialize .method state directory and STATE.json
    status                           Display current CAS state machine status
    define <TASK_ID> <CONTRACT_HASH> <BASE_COMMIT>
                                     Freeze task contract into defined state
    build                            Transition state to executing (Build stage)
    verify <DIFF_HASH> <VERIFY_LOG>  Record AROP verification and seal verify_hash
    handoff                          CAS-gated transition to handoff_complete
    recover                          Zero-token crash recovery from WAL.log
    override <OPERATOR> <RATIONALE>  Apply Master Override with hashed rationale
    diff-hash                        Compute SHA-256 of git diff --staged (<0.1ms)
    anchor-wal                       Anchor WAL head hash into git tag read-wal-head
    verify-wal-anchor                Verify WAL head hash against git tag read-wal-head
    ast-context <TARGET_FILE>        Deterministically extract C0 AST context ($0.00)
    mutate <DIFF_FILE>               Generate <=10 targeted diff mutants (<4.5s)
    eval-mutation-score <K> <TOTAL>  Evaluate mutation score against strict 100% gate
    learn <ACTION> [SCENARIO_ID]     Adversarial developer onboarding (init|scenario|diagnose|progress)
    config                           Display resolved configuration
    --version, -v                    Print engine version
    --help, -h                       Print this help message
"#
    );
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        print_usage();
        return;
    }

    let repo_root = Path::new(".");
    let engine = StateEngine::new(repo_root);

    match args[1].as_str() {
        "--version" | "-v" => {
            println!("READ Core Computational Engine (v0.7.0)");
        }
        "--help" | "-h" | "help" => {
            print_usage();
        }
        "config" => {
            let cfg = ReadConfig::load(repo_root);
            println!("{}", serde_json::to_string_pretty(&cfg).unwrap());
        }
        "init" => match engine.init() {
            Ok(_) => println!("[READ] Initialized state engine in .method/"),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "status" => match engine.load() {
            Ok(st) => println!("{}", serde_json::to_string_pretty(&st).unwrap()),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "define" => {
            if args.len() < 5 {
                eprintln!("[ERROR] Usage: read-engine define <TASK_ID> <CONTRACT_HASH> <BASE_COMMIT>");
                std::process::exit(1);
            }
            match engine.transition_define(&args[2], &args[3], &args[4]) {
                Ok(st) => println!("[READ] Task defined. State revision: {}", st.state_revision),
                Err(e) => {
                    eprintln!("[ERROR] {}", e);
                    std::process::exit(1);
                }
            }
        }
        "build" => match engine.transition_start_build() {
            Ok(st) => println!("[READ] Build started. Status: {:?}", st.status),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "verify" => {
            if args.len() < 4 {
                eprintln!("[ERROR] Usage: read-engine verify <DIFF_HASH> <VERIFY_LOG>");
                std::process::exit(1);
            }
            let log_bytes = args[3].as_bytes();
            match engine.transition_record_verification(&args[2], log_bytes) {
                Ok(st) => println!(
                    "[READ] Verification recorded. verify_hash: {}",
                    st.verify_hash.unwrap_or_default()
                ),
                Err(e) => {
                    eprintln!("[ERROR] {}", e);
                    std::process::exit(1);
                }
            }
        }
        "handoff" => match engine.transition_handoff() {
            Ok(st) => println!("[READ] CAS Handoff successful. State: {:?}", st.status),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "recover" => {
            let diff_hash = DiffHasher::hash_staged_diff(repo_root).ok().map(|(h, _)| h);
            match engine.recover(diff_hash.as_deref()) {
                Ok(msg) => println!("{}", msg),
                Err(e) => {
                    eprintln!("[ERROR] {}", e);
                    std::process::exit(1);
                }
            }
        }
        "override" => {
            if args.len() < 4 {
                eprintln!("[ERROR] Usage: read-engine override <OPERATOR> <RATIONALE>");
                std::process::exit(1);
            }
            match engine.master_override(&args[2], &args[3]) {
                Ok(st) => println!(
                    "[READ] Master Override applied. Sealed hash: {}",
                    st.verify_hash.unwrap_or_default()
                ),
                Err(e) => {
                    eprintln!("[ERROR] {}", e);
                    std::process::exit(1);
                }
            }
        }
        "diff-hash" => match DiffHasher::hash_staged_diff(repo_root) {
            Ok((hash, _)) => println!("{}", hash),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "anchor-wal" => match engine.anchor_wal_in_git() {
            Ok(head_hash) => println!("[READ] WAL head anchored to git tag read-wal-head: {}", head_hash),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "verify-wal-anchor" => match engine.verify_wal_anchor() {
            Ok(_) => println!("[READ] WAL git tag anchor verified successfully"),
            Err(e) => {
                eprintln!("[ERROR] {}", e);
                std::process::exit(1);
            }
        },
        "ast-context" => {
            if args.len() < 3 {
                eprintln!("[ERROR] Usage: read-engine ast-context <TARGET_FILE>");
                std::process::exit(1);
            }
            match AstContextResolver::resolve(&args[2], repo_root, 3) {
                Ok(manifest) => {
                    println!("[READ C0 CONTEXT MANIFEST]");
                    println!("Target: {}", manifest.target_file);
                    println!("Dependencies: {:?}", manifest.discovered_dependencies);
                    println!("Extracted Symbols ({}):", manifest.symbols.len());
                    for s in manifest.symbols {
                        println!("  {} {} -> {}", s.symbol_type, s.name, s.signature);
                    }
                    println!("Estimated context tokens: {}", manifest.estimated_token_footprint);
                }
                Err(e) => {
                    eprintln!("[ERROR] {}", e);
                    std::process::exit(1);
                }
            }
        }
        "mutate" => {
            let as_json = args.iter().any(|a| a == "--json");
            let mutator = BoundedMutator::new();
            let diff_file_arg = args.iter().skip(2).find(|a| !a.starts_with("--"));
            let mutants = if let Some(file_path) = diff_file_arg {
                let diff_content = match std::fs::read_to_string(file_path) {
                    Ok(c) => c,
                    Err(e) => {
                        eprintln!("[ERROR] Failed to read diff file: {}", e);
                        std::process::exit(1);
                    }
                };
                mutator.generate_diff_mutants(&diff_content)
            } else {
                match mutator.generate_from_staged_diff(repo_root) {
                    Ok(m) => m,
                    Err(e) => {
                        eprintln!("[ERROR] Failed to generate mutants from staged diff: {}", e);
                        std::process::exit(1);
                    }
                }
            };
            if as_json {
                println!("{}", serde_json::to_string_pretty(&mutants).unwrap());
            } else {
                println!("[READ BOUNDED DIFF MUTANTS: {}]", mutants.len());
                for m in mutants {
                    println!("  [Mutant #{}] L{}: {} -> {}", m.id, m.line_number, m.operator, m.mutated_line);
                }
            }
        }
        "eval-mutation-score" => {
            if args.len() < 4 {
                eprintln!("[ERROR] Usage: read-engine eval-mutation-score <KILLED> <TOTAL>");
                std::process::exit(1);
            }
            let killed: usize = args[2].parse().unwrap_or(0);
            let total: usize = args[3].parse().unwrap_or(0);
            let report = read_core::mutation::calculate_mutation_score(killed, total);
            if !report.passes_gate() {
                eprintln!(
                    "[AROP GATE FAILED] Mutation score {:.1}% is below strictly required 100.0% threshold (Killed {}/{}). {} mutant(s) survived. Any surviving mutant triggers immediate abort.",
                    report.kill_score_percentage, killed, total, report.surviving_mutants
                );
                std::process::exit(1);
            }
            println!(
                "[AROP GATE PASSED] Mutation score: 100.0% ({}/{} killed, 0 survivors)",
                killed, total
            );
        }
        "learn" => {
            let learn_engine = read_core::LearnEngine::new(repo_root);
            if args.len() < 3 {
                eprintln!("[ERROR] Usage: read-engine learn <init|scenario|diagnose|progress> [SCENARIO_ID]");
                std::process::exit(1);
            }
            match args[2].as_str() {
                "init" => match learn_engine.init() {
                    Ok(msg) => println!("{}", msg),
                    Err(e) => {
                        eprintln!("[ERROR] {}", e);
                        std::process::exit(1);
                    }
                },
                "scenario" => {
                    let id: usize = args.get(3).and_then(|s| s.parse().ok()).unwrap_or(1);
                    match learn_engine.setup_scenario(id) {
                        Ok(msg) => println!("{}", msg),
                        Err(e) => {
                            eprintln!("[ERROR] {}", e);
                            std::process::exit(1);
                        }
                    }
                }
                "diagnose" => match learn_engine.diagnose() {
                    Ok(msg) => println!("{}", msg),
                    Err(e) => {
                        eprintln!("[ERROR] {}", e);
                        std::process::exit(1);
                    }
                },
                "progress" => {
                    println!("{}", learn_engine.progress_report());
                }
                cmd => {
                    eprintln!("[ERROR] Unknown learn action: {}", cmd);
                    eprintln!("Usage: read-engine learn <init|scenario|diagnose|progress>");
                    std::process::exit(1);
                }
            }
        }
        cmd => {
            eprintln!("[ERROR] Unknown command: {}", cmd);
            print_usage();
            std::process::exit(1);
        }
    }
}
