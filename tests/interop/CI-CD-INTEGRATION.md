# CI/CD Integration Guide

This guide explains how to integrate the SCITT interoperability test suite into various CI/CD platforms and workflows.

## Table of Contents

- [Overview](#overview)
- [GitHub Actions](#github-actions)
- [GitLab CI](#gitlab-ci)
- [Jenkins](#jenkins)
- [CircleCI](#circleci)
- [Azure Pipelines](#azure-pipelines)
- [Local CI Testing](#local-ci-testing)
- [Environment Configuration](#environment-configuration)
- [Best Practices](#best-practices)

## Overview

The SCITT interoperability test suite is designed to run in CI/CD environments with:
- **Parallel execution** - Tests run concurrently for speed
- **Isolated environments** - Each test uses temporary directories and unique ports
- **Clear reporting** - JSON and text output formats
- **Artifact upload** - Test results saved for debugging

### Requirements

- **Go 1.22+** - For running test suite
- **Bun latest** - For TypeScript implementation
- **~2 minutes** - Typical execution time with `-parallel 10`
- **Ports 20000-30000** - For test servers (automatically allocated)

### CI/CD Workflow Stages

1. **Setup** - Install dependencies (Go, Bun)
2. **Build** - Compile both implementations
3. **Test** - Run interoperability tests
4. **Report** - Upload artifacts and generate reports
5. **Validate** - Check results and set status

## GitHub Actions

### Complete Workflow

Our repository includes `.github/workflows/ci-interop.yml`:

```yaml
name: CI - Interoperability

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  interop-tests:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]

    steps:
      - uses: actions/checkout@v4

      - name: Setup Bun
        uses: oven-sh/setup-bun@v1
        with:
          bun-version: latest

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install TypeScript dependencies
        working-directory: ./scitt-typescript
        run: bun install

      - name: Build Go CLI binary
        working-directory: ./scitt-golang
        run: |
          go build -v -o scitt ./cmd/scitt
          chmod +x scitt
          ./scitt --version || echo "Go CLI built successfully"

      - name: Build implementations
        working-directory: ./tests/interop
        run: ./scripts/build_impls.sh

      - name: Run interoperability tests
        working-directory: ./tests/interop
        run: |
          echo "Running integration test suite..."
          go test -v -parallel 10 ./...
        env:
          SCITT_GO_CLI: ${{ github.workspace }}/scitt-golang/scitt
          SCITT_TS_CLI: "bun run ${{ github.workspace }}/scitt-typescript/src/cli/index.ts"

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results-${{ matrix.os }}
          path: |
            tests/interop/**/*.log
            tests/interop/test-results.json
          retention-days: 30

      - name: Generate test report
        if: always()
        working-directory: ./tests/interop
        run: |
          go test -json ./... > test-results.json 2>&1 || true
          echo "Test results saved to test-results.json"
```

### Key Features

- **Matrix builds** - Tests on Ubuntu and macOS
- **Artifact upload** - Results saved even on failure (`if: always()`)
- **Environment variables** - CLI paths configured via `SCITT_GO_CLI` and `SCITT_TS_CLI`
- **JSON output** - Machine-readable results for analysis

### Customization

```yaml
# Add Windows testing
matrix:
  os: [ubuntu-latest, macos-latest, windows-latest]

# Add Go version matrix
strategy:
  matrix:
    os: [ubuntu-latest]
    go-version: ['1.22', '1.23']

# Add timeout
- name: Run interoperability tests
  timeout-minutes: 5
  run: go test -v -timeout 4m -parallel 10 ./...

# Add coverage
- name: Run tests with coverage
  run: go test -v -parallel 10 -coverprofile=coverage.out ./...

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v3
  with:
    files: ./tests/interop/coverage.out
```

## GitLab CI

### `.gitlab-ci.yml`

```yaml
stages:
  - build
  - test
  - report

variables:
  GO_VERSION: "1.22"
  BUN_VERSION: "latest"

# Build stage
build-go:
  stage: build
  image: golang:${GO_VERSION}
  script:
    - cd scitt-golang
    - go build -v -o scitt ./cmd/scitt
    - chmod +x scitt
  artifacts:
    paths:
      - scitt-golang/scitt
    expire_in: 1 hour

build-typescript:
  stage: build
  image: oven/bun:${BUN_VERSION}
  script:
    - cd scitt-typescript
    - bun install
  artifacts:
    paths:
      - scitt-typescript/node_modules
    expire_in: 1 hour

# Test stage
interop-tests:
  stage: test
  image: golang:${GO_VERSION}
  dependencies:
    - build-go
    - build-typescript
  before_script:
    # Install Bun in Go container
    - curl -fsSL https://bun.sh/install | bash
    - export PATH="$HOME/.bun/bin:$PATH"
  script:
    - cd tests/interop
    - export SCITT_GO_CLI="${CI_PROJECT_DIR}/scitt-golang/scitt"
    - export SCITT_TS_CLI="bun run ${CI_PROJECT_DIR}/scitt-typescript/src/cli/index.ts"
    - go test -v -parallel 10 ./...
  artifacts:
    when: always
    paths:
      - tests/interop/**/*.log
      - tests/interop/test-results.json
    expire_in: 30 days
    reports:
      junit: tests/interop/test-results.xml

# Report stage
generate-report:
  stage: report
  when: always
  script:
    - cd tests/interop
    - go test -json ./... > test-results.json 2>&1 || true
  artifacts:
    paths:
      - tests/interop/test-results.json
```

### Parallel Testing

```yaml
interop-tests:
  stage: test
  parallel:
    matrix:
      - TEST_SUITE: ["cli", "http", "crypto"]
  script:
    - cd tests/interop
    - go test -v -parallel 10 ./${TEST_SUITE}/
```

## Jenkins

### `Jenkinsfile`

```groovy
pipeline {
    agent any

    tools {
        go 'Go 1.22'
    }

    environment {
        SCITT_GO_CLI = "${WORKSPACE}/scitt-golang/scitt"
        SCITT_TS_CLI = "bun run ${WORKSPACE}/scitt-typescript/src/cli/index.ts"
    }

    stages {
        stage('Setup') {
            steps {
                // Install Bun
                sh 'curl -fsSL https://bun.sh/install | bash'
                sh 'export PATH="$HOME/.bun/bin:$PATH"'
            }
        }

        stage('Build Go') {
            steps {
                dir('scitt-golang') {
                    sh 'go build -v -o scitt ./cmd/scitt'
                    sh 'chmod +x scitt'
                    sh './scitt --version || echo "Go CLI built"'
                }
            }
        }

        stage('Build TypeScript') {
            steps {
                dir('scitt-typescript') {
                    sh 'bun install'
                }
            }
        }

        stage('Run Tests') {
            steps {
                dir('tests/interop') {
                    sh '''
                        export PATH="$HOME/.bun/bin:$PATH"
                        ./scripts/build_impls.sh
                        go test -v -parallel 10 -timeout 5m ./...
                    '''
                }
            }
        }

        stage('Generate Report') {
            steps {
                dir('tests/interop') {
                    sh 'go test -json ./... > test-results.json 2>&1 || true'
                }
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'tests/interop/**/*.log,tests/interop/test-results.json', allowEmptyArchive: true

            // Publish test results
            junit 'tests/interop/test-results.xml'
        }

        failure {
            emailext (
                subject: "Interop Tests Failed: ${env.JOB_NAME} - ${env.BUILD_NUMBER}",
                body: "Check console output at ${env.BUILD_URL}",
                recipientProviders: [culprits(), requestor()]
            )
        }
    }
}
```

### Multi-branch Pipeline

```groovy
pipeline {
    agent any

    stages {
        stage('Interop Tests') {
            when {
                anyOf {
                    branch 'main'
                    branch 'develop'
                    branch pattern: 'release/.*', comparator: 'REGEXP'
                }
            }
            steps {
                // Run tests as above
            }
        }
    }
}
```

## CircleCI

### `.circleci/config.yml`

```yaml
version: 2.1

executors:
  go-executor:
    docker:
      - image: cimg/go:1.22
    resource_class: medium

jobs:
  build-and-test:
    executor: go-executor
    steps:
      - checkout

      - run:
          name: Install Bun
          command: |
            curl -fsSL https://bun.sh/install | bash
            echo 'export PATH="$HOME/.bun/bin:$PATH"' >> $BASH_ENV

      - run:
          name: Build Go implementation
          command: |
            cd scitt-golang
            go build -v -o scitt ./cmd/scitt
            chmod +x scitt

      - run:
          name: Install TypeScript dependencies
          command: |
            cd scitt-typescript
            bun install

      - run:
          name: Run interoperability tests
          command: |
            cd tests/interop
            export SCITT_GO_CLI="${PWD}/../../scitt-golang/scitt"
            export SCITT_TS_CLI="bun run ${PWD}/../../scitt-typescript/src/cli/index.ts"
            go test -v -parallel 10 ./...

      - run:
          name: Generate test report
          when: always
          command: |
            cd tests/interop
            go test -json ./... > test-results.json 2>&1 || true

      - store_artifacts:
          path: tests/interop
          destination: test-results

      - store_test_results:
          path: tests/interop/test-results.xml

workflows:
  version: 2
  build-and-test:
    jobs:
      - build-and-test:
          filters:
            branches:
              only:
                - main
                - develop
                - /release\/.*/
```

## Azure Pipelines

### `azure-pipelines.yml`

```yaml
trigger:
  branches:
    include:
      - main
      - develop
      - release/*

pool:
  vmImage: 'ubuntu-latest'

variables:
  GO_VERSION: '1.22'
  SCITT_GO_CLI: '$(Build.SourcesDirectory)/scitt-golang/scitt'
  SCITT_TS_CLI: 'bun run $(Build.SourcesDirectory)/scitt-typescript/src/cli/index.ts'

stages:
  - stage: Build
    jobs:
      - job: BuildImplementations
        steps:
          - task: GoTool@0
            inputs:
              version: $(GO_VERSION)

          - script: |
              curl -fsSL https://bun.sh/install | bash
              export PATH="$HOME/.bun/bin:$PATH"
              bun --version
            displayName: 'Install Bun'

          - script: |
              cd scitt-golang
              go build -v -o scitt ./cmd/scitt
              chmod +x scitt
              ./scitt --version || echo "Go CLI built successfully"
            displayName: 'Build Go implementation'

          - script: |
              export PATH="$HOME/.bun/bin:$PATH"
              cd scitt-typescript
              bun install
            displayName: 'Install TypeScript dependencies'

  - stage: Test
    dependsOn: Build
    jobs:
      - job: InteropTests
        steps:
          - script: |
              export PATH="$HOME/.bun/bin:$PATH"
              cd tests/interop
              ./scripts/build_impls.sh
              go test -v -parallel 10 -timeout 5m ./...
            displayName: 'Run interoperability tests'
            env:
              SCITT_GO_CLI: $(SCITT_GO_CLI)
              SCITT_TS_CLI: $(SCITT_TS_CLI)

          - script: |
              cd tests/interop
              go test -json ./... > test-results.json 2>&1 || true
            condition: always()
            displayName: 'Generate test report'

          - task: PublishTestResults@2
            condition: always()
            inputs:
              testResultsFormat: 'JUnit'
              testResultsFiles: 'tests/interop/test-results.xml'
              failTaskOnFailedTests: true

          - task: PublishBuildArtifacts@1
            condition: always()
            inputs:
              pathToPublish: 'tests/interop'
              artifactName: 'test-results'
```

## Local CI Testing

Test your CI configuration locally before pushing:

### Act (GitHub Actions)

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | bash

# Run workflow locally
cd /path/to/transparency-service
act -j interop-tests

# Run specific job
act -j interop-tests --matrix os:ubuntu-latest

# Use larger runner
act -j interop-tests --container-architecture linux/amd64
```

### GitLab CI Local Executor

```bash
# Install gitlab-runner
brew install gitlab-runner  # macOS

# Run pipeline locally
gitlab-runner exec docker interop-tests
```

### Docker Compose for Local CI

Create `docker-compose.ci.yml`:

```yaml
version: '3.8'

services:
  ci-test:
    image: golang:1.22
    working_dir: /workspace
    volumes:
      - .:/workspace
    command: >
      bash -c "
        curl -fsSL https://bun.sh/install | bash &&
        export PATH=\"\$HOME/.bun/bin:\$PATH\" &&
        cd scitt-golang &&
        go build -v -o scitt ./cmd/scitt &&
        cd ../scitt-typescript &&
        bun install &&
        cd ../tests/interop &&
        export SCITT_GO_CLI=/workspace/scitt-golang/scitt &&
        export SCITT_TS_CLI='bun run /workspace/scitt-typescript/src/cli/index.ts' &&
        ./scripts/build_impls.sh &&
        go test -v -parallel 10 ./...
      "
```

Run with:
```bash
docker-compose -f docker-compose.ci.yml up --abort-on-container-exit
```

## Environment Configuration

### Required Environment Variables

```bash
# Go CLI binary path (absolute)
SCITT_GO_CLI=/absolute/path/to/scitt-golang/scitt

# TypeScript CLI command (with runtime)
SCITT_TS_CLI="bun run /absolute/path/to/scitt-typescript/src/cli/index.ts"
```

### Optional Environment Variables

```bash
# Test timeout (default: 120 seconds)
TEST_TIMEOUT=300

# Parallelism level (default: 10)
TEST_PARALLEL=20

# Working directory for tests
TEST_WORKDIR=/tmp/scitt-tests

# Enable verbose output
VERBOSE=1

# Skip specific test suites
SKIP_CLI_TESTS=1
SKIP_HTTP_TESTS=1
SKIP_CRYPTO_TESTS=1
```

### CI-Specific Variables

#### GitHub Actions
```yaml
env:
  SCITT_GO_CLI: ${{ github.workspace }}/scitt-golang/scitt
  SCITT_TS_CLI: "bun run ${{ github.workspace }}/scitt-typescript/src/cli/index.ts"
```

#### GitLab CI
```yaml
variables:
  SCITT_GO_CLI: "${CI_PROJECT_DIR}/scitt-golang/scitt"
  SCITT_TS_CLI: "bun run ${CI_PROJECT_DIR}/scitt-typescript/src/cli/index.ts"
```

#### Jenkins
```groovy
environment {
    SCITT_GO_CLI = "${WORKSPACE}/scitt-golang/scitt"
    SCITT_TS_CLI = "bun run ${WORKSPACE}/scitt-typescript/src/cli/index.ts"
}
```

## Best Practices

### 1. Parallel Execution

```bash
# Use parallel flag for speed
go test -v -parallel 10 ./...

# Adjust based on CI resources
# Small runners: -parallel 5
# Large runners: -parallel 20
```

### 2. Timeout Configuration

```bash
# Set reasonable timeout
go test -v -timeout 5m -parallel 10 ./...

# For slow CI environments
go test -v -timeout 10m -parallel 5 ./...
```

### 3. Artifact Collection

Always collect artifacts on failure:

```yaml
# GitHub Actions
- name: Upload test results
  if: always()  # Run even on failure
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: tests/interop/**/*.log
```

### 4. Matrix Testing

Test on multiple platforms:

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]
    go-version: ['1.22', '1.23']
```

### 5. Cache Dependencies

```yaml
# GitHub Actions - Cache Go modules
- uses: actions/cache@v3
  with:
    path: |
      ~/go/pkg/mod
      ~/.cache/go-build
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

# Cache Bun dependencies
- uses: actions/cache@v3
  with:
    path: ~/.bun/install/cache
    key: ${{ runner.os }}-bun-${{ hashFiles('**/bun.lockb') }}
```

### 6. Fail Fast

```yaml
strategy:
  fail-fast: false  # Continue testing other matrix jobs on failure
```

### 7. Status Checks

Require CI to pass before merging:

```yaml
# GitHub branch protection
require_status_checks:
  strict: true
  contexts:
    - "interop-tests (ubuntu-latest)"
    - "interop-tests (macos-latest)"
```

### 8. Notifications

```yaml
# GitHub Actions - Slack notification on failure
- name: Slack notification
  if: failure()
  uses: rtCamp/action-slack-notify@v2
  env:
    SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
    SLACK_MESSAGE: 'Interop tests failed'
```

## Monitoring and Reporting

### Test Result Trends

```bash
# Generate JSON results
go test -json ./... > test-results.json

# Parse for trends
cat test-results.json | jq -r 'select(.Action=="pass") | .Package' | sort | uniq -c
```

### Coverage Tracking

```bash
# Generate coverage
go test -coverprofile=coverage.out ./...

# Convert to HTML
go tool cover -html=coverage.out -o coverage.html

# Upload to Codecov
curl -Os https://uploader.codecov.io/latest/linux/codecov
chmod +x codecov
./codecov -f coverage.out
```

### Performance Tracking

```bash
# Run with benchmarks
go test -bench=. -benchmem ./...

# Save results
go test -bench=. -benchmem ./... | tee benchmark-results.txt
```

## Troubleshooting CI Issues

### Issue 1: Bun Not Found

```bash
# Ensure Bun is in PATH
export PATH="$HOME/.bun/bin:$PATH"

# Verify installation
bun --version
```

### Issue 2: Go Binary Not Executable

```bash
# Set execute permission
chmod +x scitt-golang/scitt

# Verify
./scitt-golang/scitt --version
```

### Issue 3: Port Conflicts

```bash
# CI environments may have port restrictions
# Tests use ports 20000-30000 by default
# Ensure this range is available
```

### Issue 4: Timeout in CI

```bash
# Increase timeout
go test -v -timeout 10m ./...

# Reduce parallelism
go test -v -parallel 5 ./...
```

## Support

For CI/CD integration issues:

1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
2. Review [ARCHITECTURE.md](./ARCHITECTURE.md) for test design
3. Consult your CI platform's documentation
4. Open an issue with CI logs

## References

- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [GitLab CI/CD Docs](https://docs.gitlab.com/ee/ci/)
- [Jenkins Pipeline Docs](https://www.jenkins.io/doc/book/pipeline/)
- [CircleCI Docs](https://circleci.com/docs/)
- [Azure Pipelines Docs](https://docs.microsoft.com/en-us/azure/devops/pipelines/)
