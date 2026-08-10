# Contributing to devboost

Thank you for your interest in contributing to devboost! This document provides guidelines and instructions for contributing.

## How to Contribute

We welcome contributions of all kinds:

- 🐛 Bug reports
- 💡 Feature requests
- 📝 Documentation improvements
- 🔧 Code contributions
- 🧪 Testing on different platforms
- 📦 New modules

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/devboost.git
   cd devboost
   ```
3. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```
4. **Build and confirm the baseline works**:
   ```bash
   go build ./...
   go test ./... -short
   ```

## Development Workflow

### 1. Make Your Changes

- Follow the coding standards in [AGENTS.md](AGENTS.md)
- Keep changes focused and atomic
- Add comments explaining *why*, not *what*
- If you're changing which tool a module uses or its defaults, research
  the choice first and document the rationale in the module's doc comment
  — see [.agents/skills/devboost-module-author/SKILL.md](.agents/skills/devboost-module-author/SKILL.md)
  for the process

### 2. Test Your Changes

**Testing is mandatory** — all contributions must be tested before submission.

#### Basic Testing Checklist

- [ ] **Build**: `go build ./...` succeeds
- [ ] **Vet**: `go vet ./...` passes
- [ ] **Unit tests**: `go test ./... -short` passes
- [ ] **Full tests** (including the slow real end-to-end apply): `go test ./...` passes
- [ ] **Plan test**: `go run ./cmd/devboost plan` shows expected changes
- [ ] **Idempotency test**: run `apply` twice — the second run should make no `.zshrc`/config changes
- [ ] **Config test**: test with a minimal config and a full config (see `.devboost.yaml.example`)

If you're adding platform-specific code, note in your PR which platforms
you were able to test on — `go test ./...` skips the platform-dependent
end-to-end test on unsupported OSes automatically.

See [tests/README.md](tests/README.md) for detailed testing documentation.

#### Module Testing

If you're adding a new module:

- [ ] Test that `Foo(cfg)` returns no resources when disabled via config
- [ ] Test the default resource shape against a default/fixture config
- [ ] Test idempotency — a resource's `Diff()` should return `nil` once converged
- [ ] Test any `DependsOn` interactions with other modules that touch the same file
- [ ] Write the rationale doc comment (see [AGENTS.md](AGENTS.md) and the module-author skill)

### 3. Update Documentation

- [ ] Update `README.md` if adding user-facing features
- [ ] Update `.devboost.yaml.example` if adding config options
- [ ] Update `CHANGELOG.md` with your changes
- [ ] Update `AGENTS.md` if changing development guidelines
- [ ] Update `ARCHITECTURE.md` if changing the engine or module structure

### 4. Commit Your Changes

Follow the [CBEAMS commit message style](AGENTS.md#4-commit-message-style-cbeams):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Examples:**

```
feat(mise): add support for a custom deno version pin

Allow toolchains.globals.deno to be set to a specific version
instead of only "lts". Defaults to "lts" if not specified.

Closes #42
```

```
fix(git): correct delta.line-numbers config key

The key was being read as git.delta.lineNumbers, which never
matched a real config key — changed to git.delta.line_numbers
to match the documented example.

Fixes #38
```

```
test(modules): add cross-module ordering regression test

Verified security's managed block correctly depends on zsh's
File resource having already run, so the file overwrite can't
silently destroy the block.

Related to #15
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub with:
- Clear description of changes
- Reference to any related issues
- Confirmation that all tests pass
- Screenshots/logs if applicable

## Adding a New Module

Adding modules is designed to be easy. See
[AGENTS.md](AGENTS.md#3-module-development) and
[ARCHITECTURE.md](ARCHITECTURE.md#adding-a-new-module) for detailed
instructions.

**Quick checklist:**

1. Create `engine/modules/foo.go` with a `Foo(cfg *config.Config) []engine.Resource` function
2. Research the tool choice and write the rationale as a doc comment (see the module-author skill)
3. Add `foo_test.go`
4. Register it in `engine/modules/registry.go`'s `All` slice
5. Add config keys to `.devboost.yaml.example`
6. `go build ./... && go test ./...`
7. Update documentation

## Code Quality Standards

- **Go idioms**: standard library over reinventing helpers, clear error wrapping (`fmt.Errorf("...: %w", err)`)
- **Error handling**: always check return values; a `CommandGuarded` with no registered implementation fails loudly, never silently
- **Readability**: since most of this code is read (and often written) by both humans and coding agents, prioritize clarity over cleverness
- **Security**: never execute unsanitized user input, validate paths, back up before overwriting a file the user didn't ask devboost to fully own
- **No corners**: never shell out to bypass writing a real diff — see [ARCHITECTURE.md](ARCHITECTURE.md#resource-kinds-providers)

## Testing Requirements

**All code must be tested before submission.**

### Minimum Testing Requirements

1. **Build and vet**: `go build ./... && go vet ./...`
2. **Unit tests**: `go test ./... -short`
3. **Full tests**: `go test ./...` (includes a real end-to-end apply against a sandboxed `HOME` — slow, touches real package managers)
4. **Idempotency**: run `apply` twice against a temp `HOME`, second run should be a no-op

### Recommended Testing

- Test with different config files
- Test error conditions (missing dependencies, unregistered `CommandGuarded` IDs, etc.)
- Test on different operating systems if possible

### Testing on Different Platforms

We especially welcome contributions that test and fix issues on:
- Different Linux distributions (Ubuntu, Debian, Fedora, Arch)
- Different macOS versions

If you test on a platform, please note it in your PR!

## Reporting Bugs

When reporting bugs, please include:

1. **Environment**:
   - OS and version
   - `devboost --version`

2. **Steps to reproduce**:
   - Exact commands run
   - Config file (if applicable)

3. **Expected behavior**:
   - What should happen

4. **Actual behavior**:
   - What actually happened
   - Error messages

5. **Additional context**:
   - Any relevant system information
   - Related issues

## Requesting Features

When requesting features:

1. **Describe the use case**: Why is this feature needed?
2. **Propose a solution**: How should it work?
3. **Consider alternatives**: Are there other ways to achieve this?
4. **Check existing issues**: Has this been requested before?

## Code Review Process

1. All PRs require review before merging
2. Reviewers will check:
   - Code quality and style
   - Test coverage
   - Documentation updates (including module rationale, if applicable)
   - Backwards compatibility
3. Be open to feedback and suggestions
4. Address review comments promptly

## Questions?

- Check [AGENTS.md](AGENTS.md) for development guidelines
- Check [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- Check [.agents/skills/devboost-module-author/SKILL.md](.agents/skills/devboost-module-author/SKILL.md) for the module-research-and-documentation process
- Open an issue for questions or discussions
- Be respectful and constructive in all interactions

## Recognition

Contributors will be:
- Listed in the README (if desired)
- Credited in release notes
- Appreciated by the community! 🎉

Thank you for contributing to devboost!
