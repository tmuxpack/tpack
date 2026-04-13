#!/usr/bin/env bats

setup() {
    export SCRIPT="$BATS_TEST_DIRNAME/../entrypoint.sh"
    export TEST_TMPDIR="$(mktemp -d)"
    export PATH="$TEST_TMPDIR/bin:$PATH"
    mkdir -p "$TEST_TMPDIR/bin"
}

teardown() {
    rm -rf "$TEST_TMPDIR"
}

stub_curl_success() {
    cat > "$TEST_TMPDIR/bin/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"token":"AAA_REGISTRATION_TOKEN","expires_at":"2099-01-01T00:00:00Z"}'
EOF
    chmod +x "$TEST_TMPDIR/bin/curl"
}

stub_curl_failure() {
    cat > "$TEST_TMPDIR/bin/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"message":"Bad credentials","documentation_url":"https://docs.github.com"}'
exit 0
EOF
    chmod +x "$TEST_TMPDIR/bin/curl"
}

stub_config_and_run() {
    # entrypoint invokes ./config.sh and ./run.sh relative to $RUNNER_WORKDIR,
    # so stubs must live in $RUNNER_WORKDIR (= $TEST_TMPDIR in tests).
    cat > "$TEST_TMPDIR/config.sh" <<EOF
#!/usr/bin/env bash
echo "config.sh called with: \$*" >> "$TEST_TMPDIR/calls.log"
EOF
    chmod +x "$TEST_TMPDIR/config.sh"
    cat > "$TEST_TMPDIR/run.sh" <<EOF
#!/usr/bin/env bash
echo "run.sh called" >> "$TEST_TMPDIR/calls.log"
EOF
    chmod +x "$TEST_TMPDIR/run.sh"
}

@test "fails fast when REPO_URL is unset" {
    unset REPO_URL
    export GITHUB_PAT=x
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"REPO_URL"* ]]
}

@test "fails fast when GITHUB_PAT is unset" {
    export REPO_URL=https://github.com/tmuxpack/tpack
    unset GITHUB_PAT
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"GITHUB_PAT"* ]]
}

@test "rejects REPO_URL that is not a github.com repo URL" {
    export REPO_URL=https://example.com/foo/bar
    export GITHUB_PAT=x
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"REPO_URL"* ]]
}

@test "extracts owner/repo from REPO_URL and calls GitHub API" {
    export REPO_URL=https://github.com/tmuxpack/tpack
    export GITHUB_PAT=fake_pat
    export RUNNER_NAME=test-runner
    export RUNNER_LABELS=aur-publisher
    export RUNNER_WORKDIR="$TEST_TMPDIR"
    stub_config_and_run
    cat > "$TEST_TMPDIR/bin/curl" <<'EOF'
#!/usr/bin/env bash
echo "$@" >> "$TEST_TMPDIR/curl_args.log"
echo '{"token":"AAA_REGISTRATION_TOKEN","expires_at":"2099-01-01T00:00:00Z"}'
EOF
    chmod +x "$TEST_TMPDIR/bin/curl"

    run bash "$SCRIPT"
    [ "$status" -eq 0 ]
    grep -q "repos/tmuxpack/tpack/actions/runners/registration-token" "$TEST_TMPDIR/curl_args.log"
    grep -q -- "--token AAA_REGISTRATION_TOKEN" "$TEST_TMPDIR/calls.log"
    grep -q -- "--labels aur-publisher" "$TEST_TMPDIR/calls.log"
    grep -q -- "--name test-runner" "$TEST_TMPDIR/calls.log"
    grep -q "run.sh called" "$TEST_TMPDIR/calls.log"
}

@test "aborts when GitHub API returns no token field" {
    export REPO_URL=https://github.com/tmuxpack/tpack
    export GITHUB_PAT=fake_pat
    export RUNNER_WORKDIR="$TEST_TMPDIR"
    stub_curl_failure
    stub_config_and_run
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Bad credentials"* ]] || [[ "$output" == *"registration token"* ]]
}
