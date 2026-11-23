package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectTestHarness(t *testing.T) {
	// 1. Jest detection
	pkgJest := &PackageJSON{
		Name: "jest-app",
		DevDependencies: map[string]string{
			"jest": "^29.0.0",
		},
	}
	cfg, err := DetectFromPackageJSON(pkgJest)
	if err != nil || cfg.Type != HarnessJest {
		t.Fatalf("Expected Jest detection, got type: %v, err: %v", cfg, err)
	}

	// 2. Vitest detection
	pkgVitest := &PackageJSON{
		Name: "vitest-app",
		DevDependencies: map[string]string{
			"vitest": "^1.0.0",
		},
	}
	cfg, err = DetectFromPackageJSON(pkgVitest)
	if err != nil || cfg.Type != HarnessVitest {
		t.Fatalf("Expected Vitest detection, got type: %v, err: %v", cfg, err)
	}

	// 3. Mocha detection
	pkgMocha := &PackageJSON{
		Name: "mocha-app",
		DevDependencies: map[string]string{
			"mocha": "^10.0.0",
		},
	}
	cfg, err = DetectFromPackageJSON(pkgMocha)
	if err != nil || cfg.Type != HarnessMocha {
		t.Fatalf("Expected Mocha detection, got type: %v, err: %v", cfg, err)
	}

	// 4. Node:test detection
	pkgNode := &PackageJSON{
		Name: "node-test-app",
		Scripts: map[string]string{
			"test": "node --test",
		},
	}
	cfg, err = DetectFromPackageJSON(pkgNode)
	if err != nil || cfg.Type != HarnessNodeTest {
		t.Fatalf("Expected NodeTest detection, got type: %v, err: %v", cfg, err)
	}

	// 5. Fail when no harness found
	pkgEmpty := &PackageJSON{
		Name: "empty-app",
		Dependencies: map[string]string{
			"express": "^4.18.0",
		},
	}
	cfg, err = DetectFromPackageJSON(pkgEmpty)
	if err == nil {
		t.Fatalf("Expected error for empty harness, got: %+v", cfg)
	}
	if err != ErrNoHarnessFound {
		t.Fatalf("Expected ErrNoHarnessFound, got: %v", err)
	}
}

func TestDetectJavaHarness(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Maven with JUnit 5
	pomJunit5 := filepath.Join(tmpDir, "pom.xml")
	err := os.WriteFile(pomJunit5, []byte(`<project><dependencies><dependency><groupId>org.junit.jupiter</groupId><artifactId>junit-jupiter</artifactId></dependency></dependencies></project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DetectFromMaven(pomJunit5)
	if err != nil || cfg.Type != HarnessJUnit5 || cfg.BuildTool != "maven" {
		t.Fatalf("Expected Maven JUnit 5 detection, got: %+v, err: %v", cfg, err)
	}

	// 2. Maven with TestNG
	pomTestNG := filepath.Join(tmpDir, "pom_testng.xml")
	err = os.WriteFile(pomTestNG, []byte(`<project><dependencies><dependency><groupId>org.testng</groupId><artifactId>testng</artifactId></dependency></dependencies></project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromMaven(pomTestNG)
	if err != nil || cfg.Type != HarnessTestNG {
		t.Fatalf("Expected Maven TestNG detection, got: %+v, err: %v", cfg, err)
	}

	// 3. Gradle with JUnit 5
	gradleJunit5 := filepath.Join(tmpDir, "build.gradle")
	err = os.WriteFile(gradleJunit5, []byte(`test { useJUnitPlatform() } dependencies { testImplementation 'org.junit.jupiter:junit-jupiter' }`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromGradle(gradleJunit5)
	if err != nil || cfg.Type != HarnessJUnit5 || cfg.BuildTool != "gradle" {
		t.Fatalf("Expected Gradle JUnit 5 detection, got: %+v, err: %v", cfg, err)
	}

	// 4. Fail on empty pom without test frameworks
	pomEmpty := filepath.Join(tmpDir, "pom_empty.xml")
	err = os.WriteFile(pomEmpty, []byte(`<project><dependencies><dependency><groupId>com.google.guava</groupId><artifactId>guava</artifactId></dependency></dependencies></project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = DetectFromMaven(pomEmpty)
	if err != ErrNoHarnessFound {
		t.Fatalf("Expected ErrNoHarnessFound for empty pom, got: %v", err)
	}
}

func TestGenerateJavaRunnerCommand(t *testing.T) {
	mvnCfg := &HarnessConfig{
		Type:          HarnessJUnit5,
		RunnerCommand: "mvn test",
		BuildTool:     "maven",
	}

	cmdMvn := GenerateRunnerCommand(mvnCfg, "", "com.example.service.AuthServiceTest#testAdminLogin")
	expectedMvn := "mvn test -Dtest=com.example.service.AuthServiceTest#testAdminLogin"
	if cmdMvn != expectedMvn {
		t.Fatalf("Expected %q, got %q", expectedMvn, cmdMvn)
	}

	gradleCfg := &HarnessConfig{
		Type:          HarnessJUnit5,
		RunnerCommand: "gradle test",
		BuildTool:     "gradle",
	}

	cmdGradle := GenerateRunnerCommand(gradleCfg, "", "com.example.service.AuthServiceTest#testAdminLogin")
	expectedGradle := "gradle test --tests com.example.service.AuthServiceTest.testAdminLogin"
	if cmdGradle != expectedGradle {
		t.Fatalf("Expected %q, got %q", expectedGradle, cmdGradle)
	}
}

func TestGenerateRunnerCommand(t *testing.T) {
	cfg := &HarnessConfig{
		Type:          HarnessJest,
		RunnerCommand: "npx jest --ci",
	}

	cmd := GenerateRunnerCommand(cfg, "junit", "tests/auth.test.ts")
	expected := "npx jest --ci --reporters=default --reporters=jest-junit tests/auth.test.ts"
	if cmd != expected {
		t.Fatalf("Expected %q, got %q", expected, cmd)
	}
}

func TestParseTAPOutput(t *testing.T) {
	tapOutput := `TAP version 13
1..3
ok 1 - auth should validate valid token
not ok 2 - rate limiter should drop 11th request # TODO
ok 3 - status endpoint should return 200
`
	report := ParseTAPOutput(tapOutput)
	if report.Total != 3 {
		t.Errorf("Expected Total=3, got %d", report.Total)
	}
	if report.Passed != 2 {
		t.Errorf("Expected Passed=2, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", report.Failed)
	}
	if report.Success {
		t.Errorf("Expected Success=false due to failure")
	}
	if len(report.Failures) != 1 || report.Failures[0] != "rate limiter should drop 11th request" {
		t.Errorf("Unexpected failures list: %v", report.Failures)
	}
}

func TestParseJUnitAndSurefireOutput(t *testing.T) {
	surefireXML := `<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="com.example.service.AuthServiceTest" tests="3" failures="1" errors="0" skipped="0" time="0.145">
  <testcase name="testValidAdminAccess" classname="com.example.service.AuthServiceTest" time="0.021"/>
  <testcase name="testNullUserRejected" classname="com.example.service.AuthServiceTest" time="0.015"/>
  <testcase name="testBoundaryQuotaExhaustion" classname="com.example.service.AuthServiceTest" time="0.032">
    <failure message="expected: &lt;false&gt; but was: &lt;true&gt;" type="org.opentest4j.AssertionFailedError">
      org.opentest4j.AssertionFailedError: expected: &lt;false&gt; but was: &lt;true&gt;
    </failure>
  </testcase>
</testsuite>`

	report, err := ParseSurefireXML(surefireXML)
	if err != nil {
		t.Fatalf("Failed to parse Surefire XML: %v", err)
	}
	if report.Total != 3 {
		t.Errorf("Expected Total=3, got %d", report.Total)
	}
	if report.Passed != 2 {
		t.Errorf("Expected Passed=2, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", report.Failed)
	}
	if report.Success {
		t.Errorf("Expected Success=false")
	}
	if len(report.Failures) != 1 || report.Failures[0] != "testBoundaryQuotaExhaustion: expected: <false> but was: <true>" {
		t.Errorf("Unexpected failures: %v", report.Failures)
	}
}

func TestDetectCSharpHarness(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. xUnit
	projXUnit := filepath.Join(tmpDir, "TestApp.csproj")
	err := os.WriteFile(projXUnit, []byte(`<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="xunit" Version="2.6.2" /><PackageReference Include="xunit.runner.visualstudio" Version="2.5.4" /></ItemGroup></Project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DetectFromCsproj(projXUnit)
	if err != nil || cfg.Type != HarnessXUnit || cfg.BuildTool != "dotnet" {
		t.Fatalf("Expected xUnit detection, got: %+v, err: %v", cfg, err)
	}

	// 2. NUnit
	projNUnit := filepath.Join(tmpDir, "NUnitTest.csproj")
	err = os.WriteFile(projNUnit, []byte(`<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="NUnit" Version="4.0.0" /></ItemGroup></Project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromCsproj(projNUnit)
	if err != nil || cfg.Type != HarnessNUnit {
		t.Fatalf("Expected NUnit detection, got: %+v, err: %v", cfg, err)
	}

	// 3. MSTest
	projMSTest := filepath.Join(tmpDir, "MSTestApp.csproj")
	err = os.WriteFile(projMSTest, []byte(`<Project Sdk="Microsoft.NET.Sdk"><ItemGroup><PackageReference Include="MSTest.TestFramework" Version="3.1.1" /></ItemGroup></Project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromCsproj(projMSTest)
	if err != nil || cfg.Type != HarnessMSTest {
		t.Fatalf("Expected MSTest detection, got: %+v, err: %v", cfg, err)
	}

	// 4. Fail on empty project
	projEmpty := filepath.Join(tmpDir, "Empty.csproj")
	err = os.WriteFile(projEmpty, []byte(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup></Project>`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromCsproj(projEmpty)
	if err != ErrNoHarnessFound {
		t.Fatalf("Expected ErrNoHarnessFound, got: %v", err)
	}
}

func TestGenerateCSharpRunnerCommand(t *testing.T) {
	cfg := &HarnessConfig{
		Type:          HarnessXUnit,
		RunnerCommand: "dotnet test --logger \"trx\"",
		ReporterType:  "trx",
		BuildTool:     "dotnet",
	}

	cmd1 := GenerateCSharpRunnerCommand(cfg, "TokenValidationServiceTest", "TestValidToken")
	expected1 := `dotnet test --filter "FullyQualifiedName~TokenValidationServiceTest.TestValidToken" --logger "trx"`
	if cmd1 != expected1 {
		t.Errorf("Expected %q, got %q", expected1, cmd1)
	}

	cmd2 := GenerateCSharpRunnerCommand(cfg, "TokenValidationServiceTest", "")
	expected2 := `dotnet test --filter "FullyQualifiedName~TokenValidationServiceTest" --logger "trx"`
	if cmd2 != expected2 {
		t.Errorf("Expected %q, got %q", expected2, cmd2)
	}

	cmd3 := GenerateCSharpRunnerCommand(cfg, "", "")
	expected3 := `dotnet test --logger "trx"`
	if cmd3 != expected3 {
		t.Errorf("Expected %q, got %q", expected3, cmd3)
	}
}

func TestParseTRXOutput(t *testing.T) {
	trxXML := `<?xml version="1.0" encoding="utf-8"?>
<TestRun id="4b88df8e-73c3-4d40-b44c-cb14e214d021" name="run" xmlns="http://microsoft.com/schemas/VisualStudio/TeamTest/2010">
  <ResultSummary outcome="Failed">
    <Counters total="4" executed="4" passed="3" failed="1" error="0" timeout="0" aborted="0" />
  </ResultSummary>
  <Results>
    <UnitTestResult executionId="1" testName="TestValidToken" outcome="Passed" />
    <UnitTestResult executionId="2" testName="TestNullTokenThrows" outcome="Passed" />
    <UnitTestResult executionId="3" testName="TestExpiredTokenFails" outcome="Passed" />
    <UnitTestResult executionId="4" testName="TestTamperedSignatureRejected" outcome="Failed">
      <Output>
        <ErrorMessage>Assert.Equal() Failure: Expected False, Actual True</ErrorMessage>
      </Output>
    </UnitTestResult>
  </Results>
</TestRun>`

	report, err := ParseTRXOutput(trxXML)
	if err != nil {
		t.Fatalf("Failed to parse TRX output: %v", err)
	}
	if report.Total != 4 {
		t.Errorf("Expected Total=4, got %d", report.Total)
	}
	if report.Passed != 3 {
		t.Errorf("Expected Passed=3, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", report.Failed)
	}
	if report.Success {
		t.Errorf("Expected Success=false")
	}
	if len(report.Failures) != 1 || report.Failures[0] != "TestTamperedSignatureRejected: Assert.Equal() Failure: Expected False, Actual True" {
		t.Errorf("Unexpected failures: %v", report.Failures)
	}
}

func TestDetectCppHarness(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. CMakeLists.txt with GTest
	cmakeGTest := filepath.Join(tmpDir, "CMakeLists.txt")
	err := os.WriteFile(cmakeGTest, []byte(`cmake_minimum_required(VERSION 3.14)
project(sample)
find_package(GTest REQUIRED)
include(CTest)
add_executable(test_sample test.cpp)
target_link_libraries(test_sample GTest::GTest GTest::Main)
add_test(NAME sample_test COMMAND test_sample)
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DetectFromCMake(cmakeGTest)
	if err != nil || cfg.Type != HarnessGTest || cfg.BuildTool != "cmake" {
		t.Fatalf("Expected CMake GTest detection, got: %+v, err: %v", cfg, err)
	}

	// 2. CMakeLists.txt with Catch2
	cmakeCatch2 := filepath.Join(tmpDir, "CMakeCatch2.txt")
	err = os.WriteFile(cmakeCatch2, []byte(`cmake_minimum_required(VERSION 3.14)
find_package(Catch2 REQUIRED)
add_test(NAME catch_test COMMAND tests)
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromCMake(cmakeCatch2)
	if err != nil || cfg.Type != HarnessCatch2 {
		t.Fatalf("Expected CMake Catch2 detection, got: %+v, err: %v", cfg, err)
	}

	// 3. Makefile with -lgtest
	makefileGTest := filepath.Join(tmpDir, "Makefile")
	err = os.WriteFile(makefileGTest, []byte(`test:
	g++ -o test_bin test.cpp -lgtest -lgtest_main -lpthread
	./test_bin
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromMakefile(makefileGTest)
	if err != nil || cfg.Type != HarnessGTest || cfg.BuildTool != "make" {
		t.Fatalf("Expected Makefile GTest detection, got: %+v, err: %v", cfg, err)
	}

	// 4. Fail on empty Makefile
	makefileEmpty := filepath.Join(tmpDir, "EmptyMakefile")
	err = os.WriteFile(makefileEmpty, []byte(`all:\n\techo "hello"`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromMakefile(makefileEmpty)
	if err != ErrNoHarnessFound {
		t.Fatalf("Expected ErrNoHarnessFound, got: %v", err)
	}
}

func TestGenerateCppRunnerCommand(t *testing.T) {
	cfgCMake := &HarnessConfig{
		Type:          HarnessGTest,
		RunnerCommand: "ctest --output-on-failure",
		ReporterType:  "junit",
		BuildTool:     "cmake",
	}

	cmd1 := GenerateCppRunnerCommand(cfgCMake, "./test_bin", "OrderTest")
	if cmd1 != "ctest -R OrderTest --output-on-failure" {
		t.Errorf("Expected 'ctest -R OrderTest --output-on-failure', got %q", cmd1)
	}

	cmd2 := GenerateCppRunnerCommand(cfgCMake, "./test_bin", "")
	if cmd2 != "ctest --output-on-failure" {
		t.Errorf("Expected 'ctest --output-on-failure', got %q", cmd2)
	}

	cfgMakeGTest := &HarnessConfig{
		Type:          HarnessGTest,
		RunnerCommand: "make test",
		ReporterType:  "junit",
		BuildTool:     "make",
	}

	cmd3 := GenerateCppRunnerCommand(cfgMakeGTest, "./custom_test", "OrderTest.Validate")
	if cmd3 != "./custom_test --gtest_filter=OrderTest.Validate" {
		t.Errorf("Expected './custom_test --gtest_filter=OrderTest.Validate', got %q", cmd3)
	}
}

func TestParseCatch2XML(t *testing.T) {
	catchXML := `<?xml version="1.0" encoding="UTF-8"?>
<Catch name="test_binary">
  <Group name="test_binary">
    <TestCase name="TestValidAllocation" filename="test.cpp" line="10">
      <OverallResult success="true"/>
    </TestCase>
    <TestCase name="TestNullPointerGuard" filename="test.cpp" line="25">
      <Expression success="false" type="CHECK">
        <Original>ptr != nullptr</Original>
        <Expanded>nullptr != nullptr</Expanded>
      </Expression>
      <OverallResult success="false"/>
    </TestCase>
    <OverallResults successes="1" failures="1" expectedFailures="0"/>
  </Group>
  <OverallResults successes="1" failures="1" expectedFailures="0"/>
</Catch>`

	report, err := ParseCatch2XML(catchXML)
	if err != nil {
		t.Fatalf("Failed to parse Catch2 XML: %v", err)
	}
	if report.Total != 2 {
		t.Errorf("Expected Total=2, got %d", report.Total)
	}
	if report.Passed != 1 {
		t.Errorf("Expected Passed=1, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", report.Failed)
	}
	if report.Success {
		t.Errorf("Expected Success=false")
	}
	if len(report.Failures) != 1 || !strings.Contains(report.Failures[0], "TestNullPointerGuard") {
		t.Errorf("Unexpected failures list: %v", report.Failures)
	}
}

func TestDetectRubyHarness(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Gemfile with RSpec
	gemfileRSpec := filepath.Join(tmpDir, "Gemfile")
	err := os.WriteFile(gemfileRSpec, []byte(`source 'https://rubygems.org'
gem 'rspec', '~> 3.12'
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := DetectFromGemfile(gemfileRSpec)
	if err != nil || cfg.Type != HarnessRSpec || cfg.BuildTool != "bundler" {
		t.Fatalf("Expected RSpec detection, got: %+v, err: %v", cfg, err)
	}

	// 2. Gemfile with Minitest
	gemfileMinitest := filepath.Join(tmpDir, "GemfileMinitest")
	err = os.WriteFile(gemfileMinitest, []byte(`source 'https://rubygems.org'
gem 'minitest', '~> 5.20'
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromGemfile(gemfileMinitest)
	if err != nil || cfg.Type != HarnessMinitest {
		t.Fatalf("Expected Minitest detection, got: %+v, err: %v", cfg, err)
	}

	// 3. Fail on empty Gemfile
	gemfileEmpty := filepath.Join(tmpDir, "GemfileEmpty")
	err = os.WriteFile(gemfileEmpty, []byte(`source 'https://rubygems.org'
gem 'rake'
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err = DetectFromGemfile(gemfileEmpty)
	if err != ErrNoHarnessFound {
		t.Fatalf("Expected ErrNoHarnessFound, got: %v", err)
	}
}

func TestGenerateRubyRunnerCommand(t *testing.T) {
	cfgRSpec := &HarnessConfig{
		Type:          HarnessRSpec,
		RunnerCommand: "bundle exec rspec --format json",
		ReporterType:  "rspec_json",
		BuildTool:     "bundler",
	}

	cmd1 := GenerateRubyRunnerCommand(cfgRSpec, "spec/auth_spec.rb", "42")
	if cmd1 != "bundle exec rspec spec/auth_spec.rb:42" {
		t.Errorf("Expected 'bundle exec rspec spec/auth_spec.rb:42', got %q", cmd1)
	}

	cmd2 := GenerateRubyRunnerCommand(cfgRSpec, "spec/auth_spec.rb", "")
	if cmd2 != "bundle exec rspec spec/auth_spec.rb" {
		t.Errorf("Expected 'bundle exec rspec spec/auth_spec.rb', got %q", cmd2)
	}

	cmd3 := GenerateRubyRunnerCommand(cfgRSpec, "", "")
	if cmd3 != "bundle exec rspec --format json" {
		t.Errorf("Expected 'bundle exec rspec --format json', got %q", cmd3)
	}

	cfgMinitest := &HarnessConfig{
		Type:          HarnessMinitest,
		RunnerCommand: "bundle exec rake test",
		ReporterType:  "tap",
		BuildTool:     "bundler",
	}

	cmd4 := GenerateRubyRunnerCommand(cfgMinitest, "test/test_auth.rb", "test_login")
	if cmd4 != "ruby -Itest test/test_auth.rb -n /test_login/" {
		t.Errorf("Expected 'ruby -Itest test/test_auth.rb -n /test_login/', got %q", cmd4)
	}
}

func TestParseRSpecJSON(t *testing.T) {
	rspecJSON := `{
  "version": "3.12.0",
  "examples": [
    {
      "description": "authenticates valid user",
      "full_description": "AuthService authenticates valid user",
      "status": "passed"
    },
    {
      "description": "rejects nil user",
      "full_description": "AuthService rejects nil user",
      "status": "failed",
      "exception": {
        "class": "RSpec::Expectations::ExpectationNotMetError",
        "message": "expected false got true"
      }
    }
  ],
  "summary": {
    "duration": 0.012,
    "example_count": 2,
    "failure_count": 1,
    "pending_count": 0
  }
}`

	report, err := ParseRSpecJSON(rspecJSON)
	if err != nil {
		t.Fatalf("Failed to parse RSpec JSON: %v", err)
	}
	if report.Total != 2 {
		t.Errorf("Expected Total=2, got %d", report.Total)
	}
	if report.Passed != 1 {
		t.Errorf("Expected Passed=1, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", report.Failed)
	}
	if report.Success {
		t.Errorf("Expected Success=false")
	}
	if len(report.Failures) != 1 || !strings.Contains(report.Failures[0], "AuthService rejects nil user: expected false got true") {
		t.Errorf("Unexpected failures list: %v", report.Failures)
	}
}
