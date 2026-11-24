package harness

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNoHarnessFound = errors.New("no supported test harness found in project configuration (supported: jest, vitest, mocha, node:test, junit4, junit5, testng, xunit, nunit, mstest, gtest, catch2, ctest, rspec, minitest)")
)

type HarnessType string

const (
	HarnessJest     HarnessType = "jest"
	HarnessVitest   HarnessType = "vitest"
	HarnessMocha    HarnessType = "mocha"
	HarnessNodeTest HarnessType = "node:test"
	HarnessJUnit5   HarnessType = "junit5"
	HarnessJUnit4   HarnessType = "junit4"
	HarnessTestNG   HarnessType = "testng"
	HarnessXUnit    HarnessType = "xunit"
	HarnessNUnit    HarnessType = "nunit"
	HarnessMSTest   HarnessType = "mstest"
	HarnessGTest    HarnessType = "gtest"
	HarnessCatch2   HarnessType = "catch2"
	HarnessCTest    HarnessType = "ctest"
	HarnessRSpec    HarnessType = "rspec"
	HarnessMinitest HarnessType = "minitest"
)

type PackageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type HarnessConfig struct {
	Type          HarnessType `json:"type"`
	RunnerCommand string      `json:"runner_command"`
	ReporterType  string      `json:"reporter_type"` // "default", "tap", "junit", "surefire"
	BuildTool     string      `json:"build_tool,omitempty"` // "maven", "gradle", "npm"
}

type ParsedTestReport struct {
	Passed   int      `json:"passed"`
	Failed   int      `json:"failed"`
	Total    int      `json:"total"`
	Success  bool     `json:"success"`
	Failures []string `json:"failures"`
}

// DetectTestHarness inspects workspace configuration files (pom.xml, build.gradle, package.json).
// Mechanically fails with ErrNoHarnessFound if no supported framework is detected.
func DetectTestHarness(workspaceDir string) (*HarnessConfig, error) {
	// 1. Maven project
	pomPath := filepath.Join(workspaceDir, "pom.xml")
	if fi, err := os.Stat(pomPath); err == nil && !fi.IsDir() {
		return DetectFromMaven(pomPath)
	}

	// 2. Gradle project
	gradlePath := filepath.Join(workspaceDir, "build.gradle")
	gradleKtsPath := filepath.Join(workspaceDir, "build.gradle.kts")
	if fi, err := os.Stat(gradlePath); err == nil && !fi.IsDir() {
		return DetectFromGradle(gradlePath)
	}
	if fi, err := os.Stat(gradleKtsPath); err == nil && !fi.IsDir() {
		return DetectFromGradle(gradleKtsPath)
	}

	// 3. Node.js project
	pkgPath := filepath.Join(workspaceDir, "package.json")
	if fi, err := os.Stat(pkgPath); err == nil && !fi.IsDir() {
		data, err := os.ReadFile(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read package.json: %w", err)
		}

		var pkg PackageJSON
		if err := json.Unmarshal(data, &pkg); err != nil {
			return nil, fmt.Errorf("malformed package.json: %w", err)
		}

		return DetectFromPackageJSON(&pkg)
	}

	// 4. .NET / C# project (.csproj)
	matches, _ := filepath.Glob(filepath.Join(workspaceDir, "*.csproj"))
	if len(matches) == 0 {
		subMatches, _ := filepath.Glob(filepath.Join(workspaceDir, "*", "*.csproj"))
		matches = append(matches, subMatches...)
	}
	if len(matches) > 0 {
		return DetectFromCsproj(matches[0])
	}

	// 5. C++ project (CMakeLists.txt or Makefile)
	cmakePath := filepath.Join(workspaceDir, "CMakeLists.txt")
	if fi, err := os.Stat(cmakePath); err == nil && !fi.IsDir() {
		return DetectFromCMake(cmakePath)
	}
	makefilePath := filepath.Join(workspaceDir, "Makefile")
	if fi, err := os.Stat(makefilePath); err == nil && !fi.IsDir() {
		return DetectFromMakefile(makefilePath)
	}

	// 6. Ruby project (Gemfile or Gemfile.lock)
	gemfilePath := filepath.Join(workspaceDir, "Gemfile")
	gemfileLockPath := filepath.Join(workspaceDir, "Gemfile.lock")
	if fi, err := os.Stat(gemfilePath); err == nil && !fi.IsDir() {
		return DetectFromGemfile(gemfilePath)
	}
	if fi, err := os.Stat(gemfileLockPath); err == nil && !fi.IsDir() {
		return DetectFromGemfile(gemfileLockPath)
	}

	return nil, ErrNoHarnessFound
}

// DetectFromMaven inspects pom.xml dependencies for JUnit 4, JUnit 5, or TestNG.
func DetectFromMaven(pomPath string) (*HarnessConfig, error) {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", pomPath, err)
	}
	content := string(data)

	if strings.Contains(content, "junit-jupiter") || strings.Contains(content, "org.junit.jupiter") {
		return &HarnessConfig{
			Type:          HarnessJUnit5,
			RunnerCommand: "mvn test",
			ReporterType:  "surefire",
			BuildTool:     "maven",
		}, nil
	}
	if strings.Contains(content, "junit") && strings.Contains(content, "<artifactId>junit</artifactId>") {
		return &HarnessConfig{
			Type:          HarnessJUnit4,
			RunnerCommand: "mvn test",
			ReporterType:  "surefire",
			BuildTool:     "maven",
		}, nil
	}
	if strings.Contains(content, "testng") || strings.Contains(content, "org.testng") {
		return &HarnessConfig{
			Type:          HarnessTestNG,
			RunnerCommand: "mvn test",
			ReporterType:  "surefire",
			BuildTool:     "maven",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// DetectFromGradle inspects build.gradle for JUnit 4, JUnit 5, or TestNG.
func DetectFromGradle(gradlePath string) (*HarnessConfig, error) {
	data, err := os.ReadFile(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", gradlePath, err)
	}
	content := string(data)

	if strings.Contains(content, "useJUnitPlatform") || strings.Contains(content, "junit-jupiter") {
		return &HarnessConfig{
			Type:          HarnessJUnit5,
			RunnerCommand: "gradle test",
			ReporterType:  "surefire",
			BuildTool:     "gradle",
		}, nil
	}
	if strings.Contains(content, "useTestNG") || strings.Contains(content, "testng") {
		return &HarnessConfig{
			Type:          HarnessTestNG,
			RunnerCommand: "gradle test",
			ReporterType:  "surefire",
			BuildTool:     "gradle",
		}, nil
	}
	if strings.Contains(content, "junit:junit") || strings.Contains(content, "'junit'") || strings.Contains(content, "\"junit\"") {
		return &HarnessConfig{
			Type:          HarnessJUnit4,
			RunnerCommand: "gradle test",
			ReporterType:  "surefire",
			BuildTool:     "gradle",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// GenerateJavaRunnerCommand builds targeted test command for class and optional method.
func GenerateJavaRunnerCommand(cfg *HarnessConfig, className, methodName string) string {
	if cfg.BuildTool == "gradle" {
		if className == "" {
			return "gradle test"
		}
		if methodName != "" {
			return fmt.Sprintf("gradle test --tests %s.%s", className, methodName)
		}
		return fmt.Sprintf("gradle test --tests %s", className)
	}

	// Default Maven
	if className == "" {
		return "mvn test"
	}
	if methodName != "" {
		return fmt.Sprintf("mvn test -Dtest=%s#%s", className, methodName)
	}
	return fmt.Sprintf("mvn test -Dtest=%s", className)
}

// DetectFromCsproj inspects .csproj dependencies for xUnit, NUnit, or MSTest.
func DetectFromCsproj(csprojPath string) (*HarnessConfig, error) {
	data, err := os.ReadFile(csprojPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", csprojPath, err)
	}
	content := strings.ToLower(string(data))

	if strings.Contains(content, "xunit") {
		return &HarnessConfig{
			Type:          HarnessXUnit,
			RunnerCommand: "dotnet test --logger \"trx\"",
			ReporterType:  "trx",
			BuildTool:     "dotnet",
		}, nil
	}
	if strings.Contains(content, "nunit") {
		return &HarnessConfig{
			Type:          HarnessNUnit,
			RunnerCommand: "dotnet test --logger \"trx\"",
			ReporterType:  "trx",
			BuildTool:     "dotnet",
		}, nil
	}
	if strings.Contains(content, "mstest") || strings.Contains(content, "mstest.testframework") {
		return &HarnessConfig{
			Type:          HarnessMSTest,
			RunnerCommand: "dotnet test --logger \"trx\"",
			ReporterType:  "trx",
			BuildTool:     "dotnet",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// GenerateCSharpRunnerCommand builds targeted test command for class and optional method.
func GenerateCSharpRunnerCommand(cfg *HarnessConfig, className, methodName string) string {
	filter := ""
	if className != "" {
		if methodName != "" {
			filter = fmt.Sprintf(" --filter \"FullyQualifiedName~%s.%s\"", className, methodName)
		} else {
			filter = fmt.Sprintf(" --filter \"FullyQualifiedName~%s\"", className)
		}
	}
	return fmt.Sprintf("dotnet test%s --logger \"trx\"", filter)
}

// DetectFromCMake inspects CMakeLists.txt for Google Test, Catch2, or CTest.
func DetectFromCMake(cmakePath string) (*HarnessConfig, error) {
	data, err := os.ReadFile(cmakePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", cmakePath, err)
	}
	content := strings.ToLower(string(data))

	if strings.Contains(content, "gtest") || strings.Contains(content, "googletest") {
		return &HarnessConfig{
			Type:          HarnessGTest,
			RunnerCommand: "ctest --output-on-failure",
			ReporterType:  "junit",
			BuildTool:     "cmake",
		}, nil
	}
	if strings.Contains(content, "catch2") {
		return &HarnessConfig{
			Type:          HarnessCatch2,
			RunnerCommand: "ctest --output-on-failure",
			ReporterType:  "catch2",
			BuildTool:     "cmake",
		}, nil
	}
	if strings.Contains(content, "include(ctest)") || strings.Contains(content, "enable_testing()") {
		return &HarnessConfig{
			Type:          HarnessCTest,
			RunnerCommand: "ctest --output-on-failure",
			ReporterType:  "junit",
			BuildTool:     "cmake",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// DetectFromMakefile inspects Makefile for Google Test, Catch2, or test targets.
func DetectFromMakefile(makefilePath string) (*HarnessConfig, error) {
	data, err := os.ReadFile(makefilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", makefilePath, err)
	}
	content := strings.ToLower(string(data))

	if strings.Contains(content, "gtest") || strings.Contains(content, "-lgtest") {
		return &HarnessConfig{
			Type:          HarnessGTest,
			RunnerCommand: "make test",
			ReporterType:  "junit",
			BuildTool:     "make",
		}, nil
	}
	if strings.Contains(content, "catch2") {
		return &HarnessConfig{
			Type:          HarnessCatch2,
			RunnerCommand: "make test",
			ReporterType:  "catch2",
			BuildTool:     "make",
		}, nil
	}
	if strings.Contains(content, "test:") || strings.Contains(content, "check:") {
		return &HarnessConfig{
			Type:          HarnessCTest,
			RunnerCommand: "make test",
			ReporterType:  "junit",
			BuildTool:     "make",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// GenerateCppRunnerCommand builds targeted C++ runner command.
func GenerateCppRunnerCommand(cfg *HarnessConfig, testBinary, testName string) string {
	if cfg.BuildTool == "cmake" || cfg.Type == HarnessCTest {
		if testName != "" {
			return fmt.Sprintf("ctest -R %s --output-on-failure", testName)
		}
		return "ctest --output-on-failure"
	}

	bin := testBinary
	if bin == "" {
		bin = "./test_binary"
	}
	if cfg.Type == HarnessGTest {
		if testName != "" {
			return fmt.Sprintf("%s --gtest_filter=%s", bin, testName)
		}
		return fmt.Sprintf("%s --gtest_output=xml:test_detail.xml", bin)
	}
	if cfg.Type == HarnessCatch2 {
		if testName != "" {
			return fmt.Sprintf("%s %s -r xml", bin, testName)
		}
		return fmt.Sprintf("%s -r xml", bin)
	}

	return bin
}

// DetectFromGemfile inspects Gemfile or Gemfile.lock for RSpec or Minitest.
func DetectFromGemfile(path string) (*HarnessConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	content := strings.ToLower(string(data))

	if strings.Contains(content, "rspec") {
		return &HarnessConfig{
			Type:          HarnessRSpec,
			RunnerCommand: "bundle exec rspec --format json",
			ReporterType:  "rspec_json",
			BuildTool:     "bundler",
		}, nil
	}
	if strings.Contains(content, "minitest") {
		return &HarnessConfig{
			Type:          HarnessMinitest,
			RunnerCommand: "bundle exec rake test",
			ReporterType:  "tap",
			BuildTool:     "bundler",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// GenerateRubyRunnerCommand builds targeted Ruby test command.
func GenerateRubyRunnerCommand(cfg *HarnessConfig, testFile, testTarget string) string {
	if cfg.Type == HarnessRSpec {
		if testFile != "" {
			if testTarget != "" {
				return fmt.Sprintf("bundle exec rspec %s:%s", testFile, testTarget)
			}
			return fmt.Sprintf("bundle exec rspec %s", testFile)
		}
		return "bundle exec rspec --format json"
	}

	// Default Minitest
	if testFile != "" {
		if testTarget != "" {
			return fmt.Sprintf("ruby -Itest %s -n /%s/", testFile, testTarget)
		}
		return fmt.Sprintf("ruby -Itest %s", testFile)
	}
	return "bundle exec rake test"
}

// DetectFromPackageJSON inspects package.json dependencies and scripts.
func DetectFromPackageJSON(pkg *PackageJSON) (*HarnessConfig, error) {
	hasDep := func(name string) bool {
		if pkg.DevDependencies != nil {
			if _, ok := pkg.DevDependencies[name]; ok {
				return true
			}
		}
		if pkg.Dependencies != nil {
			if _, ok := pkg.Dependencies[name]; ok {
				return true
			}
		}
		return false
	}

	scriptTest := ""
	if pkg.Scripts != nil {
		scriptTest = pkg.Scripts["test"]
	}

	// 1. Jest
	if hasDep("jest") || strings.Contains(scriptTest, "jest") {
		return &HarnessConfig{
			Type:          HarnessJest,
			RunnerCommand: "npx jest --ci",
			ReporterType:  "junit",
			BuildTool:     "npm",
		}, nil
	}

	// 2. Vitest
	if hasDep("vitest") || strings.Contains(scriptTest, "vitest") {
		return &HarnessConfig{
			Type:          HarnessVitest,
			RunnerCommand: "npx vitest run --reporter=tap",
			ReporterType:  "tap",
			BuildTool:     "npm",
		}, nil
	}

	// 3. Mocha
	if hasDep("mocha") || strings.Contains(scriptTest, "mocha") {
		return &HarnessConfig{
			Type:          HarnessMocha,
			RunnerCommand: "npx mocha --reporter tap",
			ReporterType:  "tap",
			BuildTool:     "npm",
		}, nil
	}

	// 4. Node:test (node --test)
	if strings.Contains(scriptTest, "node --test") || strings.Contains(scriptTest, "node:test") {
		return &HarnessConfig{
			Type:          HarnessNodeTest,
			RunnerCommand: "node --test --test-reporter=tap",
			ReporterType:  "tap",
			BuildTool:     "node",
		}, nil
	}

	return nil, ErrNoHarnessFound
}

// GenerateRunnerCommand produces the exact CLI invocation for running tests.
func GenerateRunnerCommand(cfg *HarnessConfig, reporter string, testFiles ...string) string {
	if cfg.BuildTool == "maven" || cfg.BuildTool == "gradle" {
		targetClass := ""
		targetMethod := ""
		if len(testFiles) > 0 {
			target := testFiles[0]
			if strings.Contains(target, "#") {
				parts := strings.SplitN(target, "#", 2)
				targetClass = parts[0]
				targetMethod = parts[1]
			} else if strings.Contains(target, ".") && !strings.HasSuffix(target, ".java") {
				parts := strings.Split(target, ".")
				if len(parts) >= 2 {
					targetClass = strings.Join(parts[:len(parts)-1], ".")
					targetMethod = parts[len(parts)-1]
				} else {
					targetClass = target
				}
			} else {
				targetClass = strings.TrimSuffix(filepath.Base(target), ".java")
			}
		}
		return GenerateJavaRunnerCommand(cfg, targetClass, targetMethod)
	}

	base := cfg.RunnerCommand
	if reporter != "" {
		switch cfg.Type {
		case HarnessJest:
			if reporter == "tap" {
				base = "npx jest --ci --reporters=jest-tap-reporter"
			} else if reporter == "junit" {
				base = "npx jest --ci --reporters=default --reporters=jest-junit"
			}
		case HarnessVitest:
			if reporter == "junit" {
				base = "npx vitest run --reporter=junit"
			} else if reporter == "tap" {
				base = "npx vitest run --reporter=tap"
			}
		case HarnessMocha:
			if reporter == "junit" {
				base = "npx mocha --reporter mocha-junit-reporter"
			} else if reporter == "tap" {
				base = "npx mocha --reporter tap"
			}
		case HarnessNodeTest:
			if reporter == "tap" {
				base = "node --test --test-reporter=tap"
			} else if reporter == "junit" {
				base = "node --test --test-reporter=junit"
			}
		}
	}

	if len(testFiles) > 0 {
		return fmt.Sprintf("%s %s", base, strings.Join(testFiles, " "))
	}
	return base
}

// ParseTAPOutput parses Test Anything Protocol (TAP) stream.
func ParseTAPOutput(raw string) *ParsedTestReport {
	report := &ParsedTestReport{
		Failures: make([]string, 0),
	}

	lines := strings.Split(raw, "\n")
	planRe := regexp.MustCompile(`^1\.\.(\d+)`)
	okRe := regexp.MustCompile(`^(ok|not ok)\s+(\d+)?\s*(.*?)(?:\s*#\s*(.*))?$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if matches := planRe.FindStringSubmatch(line); len(matches) > 1 {
			if count, err := strconv.Atoi(matches[1]); err == nil {
				report.Total = count
			}
			continue
		}

		if matches := okRe.FindStringSubmatch(line); len(matches) > 1 {
			status := matches[1]
			desc := strings.TrimSpace(matches[3])
			desc = strings.TrimPrefix(desc, "- ")
			if status == "ok" {
				report.Passed++
			} else {
				report.Failed++
				report.Failures = append(report.Failures, desc)
			}
		}
	}

	if report.Total == 0 {
		report.Total = report.Passed + report.Failed
	}
	report.Success = report.Failed == 0 && report.Total > 0
	return report
}

// JUnit XML Structures
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	XMLName xml.Name      `xml:"testcase"`
	Name    string        `xml:"name,attr"`
	Failure *JUnitFailure `xml:"failure"`
	Error   *JUnitFailure `xml:"error"`
}

type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

// ParseSurefireXML parses Maven Surefire / Failsafe XML report.
func ParseSurefireXML(raw string) (*ParsedTestReport, error) {
	return ParseJUnitOutput(raw)
}

// ParseJUnitOutput parses JUnit XML test report.
func ParseJUnitOutput(raw string) (*ParsedTestReport, error) {
	report := &ParsedTestReport{
		Failures: make([]string, 0),
	}

	var suites JUnitTestSuites
	err := xml.Unmarshal([]byte(raw), &suites)
	if err == nil && (suites.Tests > 0 || len(suites.TestSuites) > 0) {
		report.Total = suites.Tests
		report.Failed = suites.Failures + suites.Errors
		for _, ts := range suites.TestSuites {
			for _, tc := range ts.TestCases {
				if tc.Failure != nil {
					msg := tc.Failure.Message
					if msg == "" {
						msg = strings.TrimSpace(tc.Failure.Body)
					}
					report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", tc.Name, msg))
				} else if tc.Error != nil {
					msg := tc.Error.Message
					if msg == "" {
						msg = strings.TrimSpace(tc.Error.Body)
					}
					report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", tc.Name, msg))
				}
			}
		}
		if report.Total == 0 {
			for _, ts := range suites.TestSuites {
				report.Total += ts.Tests
			}
		}
		report.Passed = report.Total - report.Failed
		report.Success = report.Failed == 0 && report.Total > 0
		return report, nil
	}

	var singleSuite JUnitTestSuite
	if errSingle := xml.Unmarshal([]byte(raw), &singleSuite); errSingle == nil && (singleSuite.Tests > 0 || len(singleSuite.TestCases) > 0) {
		report.Total = singleSuite.Tests
		report.Failed = singleSuite.Failures + singleSuite.Errors
		for _, tc := range singleSuite.TestCases {
			if tc.Failure != nil {
				msg := tc.Failure.Message
				if msg == "" {
					msg = strings.TrimSpace(tc.Failure.Body)
				}
				report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", tc.Name, msg))
			} else if tc.Error != nil {
				msg := tc.Error.Message
				if msg == "" {
					msg = strings.TrimSpace(tc.Error.Body)
				}
				report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", tc.Name, msg))
			}
		}
		if report.Total == 0 {
			report.Total = len(singleSuite.TestCases)
		}
		report.Passed = report.Total - report.Failed
		report.Success = report.Failed == 0 && report.Total > 0
		return report, nil
	}

	return nil, fmt.Errorf("failed to parse JUnit/Surefire XML: %v", err)
}

// TRX Test Results Structures
type TRXTestRun struct {
	XMLName       xml.Name         `xml:"TestRun"`
	ResultSummary TRXResultSummary `xml:"ResultSummary"`
	Results       TRXResults       `xml:"Results"`
}

type TRXResultSummary struct {
	Outcome  string      `xml:"outcome,attr"`
	Counters TRXCounters `xml:"Counters"`
}

type TRXCounters struct {
	Total    int `xml:"total,attr"`
	Executed int `xml:"executed,attr"`
	Passed   int `xml:"passed,attr"`
	Failed   int `xml:"failed,attr"`
	Errors   int `xml:"error,attr"`
	Timeout  int `xml:"timeout,attr"`
	Aborted  int `xml:"aborted,attr"`
}

type TRXResults struct {
	UnitTestResults []TRXUnitTestResult `xml:"UnitTestResult"`
}

type TRXUnitTestResult struct {
	TestName string    `xml:"testName,attr"`
	Outcome  string    `xml:"outcome,attr"`
	Output   TRXOutput `xml:"Output"`
}

type TRXOutput struct {
	ErrorMessage string `xml:"ErrorMessage"`
	StackTrace   string `xml:"StackTrace"`
}

// ParseTRXOutput parses .NET TRX XML test results.
func ParseTRXOutput(raw string) (*ParsedTestReport, error) {
	report := &ParsedTestReport{
		Failures: make([]string, 0),
	}

	var testRun TRXTestRun
	err := xml.Unmarshal([]byte(raw), &testRun)
	if err == nil && (testRun.ResultSummary.Counters.Total > 0 || len(testRun.Results.UnitTestResults) > 0) {
		report.Total = testRun.ResultSummary.Counters.Total
		report.Passed = testRun.ResultSummary.Counters.Passed
		report.Failed = testRun.ResultSummary.Counters.Failed + testRun.ResultSummary.Counters.Errors + testRun.ResultSummary.Counters.Timeout + testRun.ResultSummary.Counters.Aborted

		for _, r := range testRun.Results.UnitTestResults {
			if strings.EqualFold(r.Outcome, "Failed") || strings.EqualFold(r.Outcome, "Error") || strings.EqualFold(r.Outcome, "Timeout") {
				msg := strings.TrimSpace(r.Output.ErrorMessage)
				if msg == "" {
					msg = strings.TrimSpace(r.Output.StackTrace)
				}
				if msg == "" {
					msg = "Test failed"
				}
				report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", r.TestName, msg))
			}
		}

		if report.Total == 0 {
			report.Total = len(testRun.Results.UnitTestResults)
		}
		if report.Passed == 0 && report.Failed == 0 && report.Total > 0 {
			report.Passed = report.Total
		}
		report.Success = report.Failed == 0 && report.Total > 0
		return report, nil
	}

	// Fallback regex matching
	reCounters := regexp.MustCompile(`total="(\d+)".*passed="(\d+)".*failed="(\d+)"`)
	if match := reCounters.FindStringSubmatch(raw); len(match) == 4 {
		total, _ := strconv.Atoi(match[1])
		passed, _ := strconv.Atoi(match[2])
		failed, _ := strconv.Atoi(match[3])
		report.Total = total
		report.Passed = passed
		report.Failed = failed
		report.Success = failed == 0 && total > 0
		return report, nil
	}

	return nil, fmt.Errorf("failed to parse TRX test results: %v", err)
}

// Catch2 XML structures
type Catch2XMLReport struct {
	XMLName        xml.Name                `xml:"Catch"`
	Group          Catch2XMLGroup          `xml:"Group"`
	OverallResults Catch2XMLOverallResults `xml:"OverallResults"`
}

type Catch2XMLGroup struct {
	TestCases      []Catch2XMLTestCase     `xml:"TestCase"`
	OverallResults Catch2XMLOverallResults `xml:"OverallResults"`
}

type Catch2XMLTestCase struct {
	Name          string                  `xml:"name,attr"`
	OverallResult Catch2XMLTestCaseResult `xml:"OverallResult"`
	Expression    []Catch2XMLExpression   `xml:"Expression"`
}

type Catch2XMLTestCaseResult struct {
	Success string `xml:"success,attr"`
}

type Catch2XMLExpression struct {
	Success  string `xml:"success,attr"`
	Type     string `xml:"type,attr"`
	Original string `xml:"Original"`
	Expanded string `xml:"Expanded"`
}

type Catch2XMLOverallResults struct {
	Successes int `xml:"successes,attr"`
	Failures  int `xml:"failures,attr"`
	Expected  int `xml:"expectedFailures,attr"`
}

// ParseCatch2XML parses Catch2 XML test reports.
func ParseCatch2XML(raw string) (*ParsedTestReport, error) {
	report := &ParsedTestReport{
		Failures: make([]string, 0),
	}

	var catchDoc Catch2XMLReport
	err := xml.Unmarshal([]byte(raw), &catchDoc)
	if err == nil && (catchDoc.OverallResults.Successes > 0 || catchDoc.OverallResults.Failures > 0 || len(catchDoc.Group.TestCases) > 0) {
		report.Passed = catchDoc.OverallResults.Successes
		report.Failed = catchDoc.OverallResults.Failures
		if report.Passed == 0 && report.Failed == 0 {
			report.Passed = catchDoc.Group.OverallResults.Successes
			report.Failed = catchDoc.Group.OverallResults.Failures
		}

		for _, tc := range catchDoc.Group.TestCases {
			if strings.EqualFold(tc.OverallResult.Success, "false") {
				msg := "Test assertion failed"
				for _, expr := range tc.Expression {
					if strings.EqualFold(expr.Success, "false") {
						msg = fmt.Sprintf("%s (expanded: %s)", expr.Original, expr.Expanded)
						break
					}
				}
				report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", tc.Name, msg))
			}
		}

		report.Total = report.Passed + report.Failed
		report.Success = report.Failed == 0 && report.Total > 0
		return report, nil
	}

	// Fallback regex matching
	reResults := regexp.MustCompile(`successes="(\d+)".*failures="(\d+)"`)
	if match := reResults.FindStringSubmatch(raw); len(match) == 3 {
		passed, _ := strconv.Atoi(match[1])
		failed, _ := strconv.Atoi(match[2])
		report.Passed = passed
		report.Failed = failed
		report.Total = passed + failed
		report.Success = failed == 0 && report.Total > 0
		return report, nil
	}

	return nil, fmt.Errorf("failed to parse Catch2 XML: %v", err)
}

// RSpec JSON structures
type RSpecJSONReport struct {
	Examples []RSpecExample `json:"examples"`
	Summary  RSpecSummary   `json:"summary"`
}

type RSpecExample struct {
	ID              string          `json:"id"`
	Description     string          `json:"description"`
	FullDescription string          `json:"full_description"`
	Status          string          `json:"status"`
	Exception       *RSpecException `json:"exception,omitempty"`
}

type RSpecException struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

type RSpecSummary struct {
	Duration     float64 `json:"duration"`
	ExampleCount int     `json:"example_count"`
	FailureCount int     `json:"failure_count"`
	PendingCount int     `json:"pending_count"`
}

// ParseRSpecJSON parses RSpec JSON test report.
func ParseRSpecJSON(raw string) (*ParsedTestReport, error) {
	report := &ParsedTestReport{
		Failures: make([]string, 0),
	}

	var rspecDoc RSpecJSONReport
	err := json.Unmarshal([]byte(raw), &rspecDoc)
	if err == nil && (rspecDoc.Summary.ExampleCount > 0 || len(rspecDoc.Examples) > 0) {
		report.Total = rspecDoc.Summary.ExampleCount
		report.Failed = rspecDoc.Summary.FailureCount
		if report.Total == 0 {
			report.Total = len(rspecDoc.Examples)
		}
		report.Passed = report.Total - report.Failed

		for _, ex := range rspecDoc.Examples {
			if strings.EqualFold(ex.Status, "failed") {
				msg := "RSpec expectation failed"
				if ex.Exception != nil && ex.Exception.Message != "" {
					msg = ex.Exception.Message
				}
				name := ex.FullDescription
				if name == "" {
					name = ex.Description
				}
				report.Failures = append(report.Failures, fmt.Sprintf("%s: %s", name, msg))
			}
		}

		report.Success = report.Failed == 0 && report.Total > 0
		return report, nil
	}

	// Fallback regex matching
	reSummary := regexp.MustCompile(`"example_count":\s*(\d+).*?"failure_count":\s*(\d+)`)
	if match := reSummary.FindStringSubmatch(raw); len(match) == 3 {
		total, _ := strconv.Atoi(match[1])
		failed, _ := strconv.Atoi(match[2])
		report.Total = total
		report.Failed = failed
		report.Passed = total - failed
		report.Success = failed == 0 && total > 0
		return report, nil
	}

	return nil, fmt.Errorf("failed to parse RSpec JSON: %v", err)
}
