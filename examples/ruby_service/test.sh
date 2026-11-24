#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

ruby - << 'EOF'
require 'json'
require 'time'

start_time = Process.clock_gettime(Process::CLOCK_MONOTONIC)

failures = []
examples = []

def run_test(id, desc, full_desc, line, examples, failures)
  begin
    yield
    examples << {
      "id" => id,
      "description" => desc,
      "full_description" => full_desc,
      "status" => "passed",
      "file_path" => "./spec/auth_service_spec.rb",
      "line_number" => line
    }
  rescue => e
    examples << {
      "id" => id,
      "description" => desc,
      "full_description" => full_desc,
      "status" => "failed",
      "file_path" => "./spec/auth_service_spec.rb",
      "line_number" => line,
      "exception" => {
        "class" => e.class.to_s,
        "message" => e.message,
        "backtrace" => e.backtrace ? e.backtrace.take(5) : []
      }
    }
    failures << "#{full_desc}: #{e.class} - #{e.message}"
  end
end

require_relative 'lib/auth_service'
validator = AuthService::TokenValidator.new('super_secure_key', 3600)

# Test 1: Nil user handling (kills REMOVE_SAFE_NAVIGATION)
run_test("./spec/auth_service_spec.rb[1:1:1]",
         "returns false when user is nil (kills REMOVE_SAFE_NAVIGATION)",
         "AuthService::TokenValidator #validate_user nil handling",
         10, examples, failures) do
  res = validator.validate_user(nil)
  raise "Expected false, got #{res.inspect}" unless res == false
end

# Test 2: Token format regex match (kills INVERT_EQUALITY ===)
run_test("./spec/auth_service_spec.rb[1:2:1]",
         "validates alphanumeric token using case equality (kills INVERT_EQUALITY ===)",
         "AuthService::TokenValidator #validate_token_format regex match",
         25, examples, failures) do
  res_valid = validator.validate_token_format("valid_token_123")
  res_invalid = validator.validate_token_format("invalid-token!")
  raise "Expected true for valid token" unless res_valid == true
  raise "Expected false for invalid token" unless res_invalid == false
end

# Test 3: Secret equality via eql? (kills INVERT_EQUALITY .eql?)
run_test("./spec/auth_service_spec.rb[1:3:1]",
         "validates secret equality via eql? (kills INVERT_EQUALITY .eql?)",
         "AuthService::TokenValidator #validate_secret eql? match",
         33, examples, failures) do
  res_match = validator.validate_secret("super_secure_key")
  res_diff = validator.validate_secret("wrong_key")
  raise "Expected true for matching secret" unless res_match == true
  raise "Expected false for mismatched secret" unless res_diff == false
end

# Test 4: Status code equality (kills INVERT_EQUALITY ==)
run_test("./spec/auth_service_spec.rb[1:4:1]",
         "validates status code 200 (kills INVERT_EQUALITY ==)",
         "AuthService::TokenValidator #validate_status status equality",
         41, examples, failures) do
  res_ok = validator.validate_status(200)
  res_err = validator.validate_status(403)
  raise "Expected true for status 200" unless res_ok == true
  raise "Expected false for status 403" unless res_err == false
end

# Test 5: TTL boundaries (kills BOUNDARY_SHIFT <)
run_test("./spec/auth_service_spec.rb[1:5:1]",
         "validates ttl lower boundary at 60 (kills BOUNDARY_SHIFT <)",
         "AuthService::TokenValidator #validate_ttl boundary check",
         49, examples, failures) do
  raise "Expected false for ttl 59" unless validator.validate_ttl(59) == false
  raise "Expected true for ttl 60" unless validator.validate_ttl(60) == true
  raise "Expected true for ttl 3600" unless validator.validate_ttl(3600) == true
  raise "Expected false for ttl 3601" unless validator.validate_ttl(3601) == false
end

# Test 6: Rate limit retry increment (kills BOUNDARY_SHIFT + 1)
run_test("./spec/auth_service_spec.rb[1:6:1]",
         "validates retry boundary increment + 1 (kills BOUNDARY_SHIFT + 1)",
         "AuthService::TokenValidator #check_rate_limit increment check",
         59, examples, failures) do
  raise "Expected true for 0 attempts" unless validator.check_rate_limit(0) == true
  raise "Expected true for 2 attempts" unless validator.check_rate_limit(2) == true
  raise "Expected false for 3 attempts" unless validator.check_rate_limit(3) == false
end

# Test 7: Session payload returned (kills RETURN_NIL)
run_test("./spec/auth_service_spec.rb[1:7:1]",
         "returns valid session payload and not nil (kills RETURN_NIL)",
         "AuthService::TokenValidator #build_session non-nil payload",
         68, examples, failures) do
  sess = validator.build_session("usr_99", "tok_abc")
  raise "Expected session not to be nil" if sess.nil?
  raise "Expected ACTIVE status" unless sess[:status] == "ACTIVE"
  raise "Expected user_id usr_99" unless sess[:user_id] == "usr_99"
end

duration = Process.clock_gettime(Process::CLOCK_MONOTONIC) - start_time

report = {
  "examples" => examples,
  "summary" => {
    "duration" => duration.round(6),
    "example_count" => examples.size,
    "failure_count" => failures.size,
    "pending_count" => 0
  }
}

File.write("rspec_report.json", JSON.pretty_generate(report))

if failures.empty?
  puts "All #{examples.size} examples passed (0 failures)"
  exit 0
else
  warn "Failures (#{failures.size}):"
  failures.each { |f| warn "  - #{f}" }
  exit 1
end
EOF
