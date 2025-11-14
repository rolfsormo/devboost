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

## Development Workflow

### 1. Make Your Changes

- Follow the coding standards in [AGENTS.md](AGENTS.md)
- Keep changes focused and atomic
- Add comments explaining *why*, not *what*
- Use the `db_*` function naming convention

### 2. Test Your Changes

**Testing is mandatory** - all contributions must be tested before submission.

#### Basic Testing Checklist

- [ ] **Build test**: `./build.sh` succeeds
- [ ] **Syntax check**: `bash -n dist/devboost.sh` passes
- [ ] **Plan test**: `./dist/devboost.sh plan` shows expected changes
- [ ] **Idempotency test**: Run `apply` twice - second run should be no-op
- [ ] **Config test**: Test with minimal config and full config
- [ ] **Dry-run test**: `./dist/devboost.sh plan` works correctly

#### Platform Testing

If you're adding platform-specific code:

- [ ] Test on macOS (if available)
- [ ] Test on Linux (Ubuntu/Debian, Fedora, or Arch if available)
- [ ] Document any platform limitations

#### Module Testing

If you're adding a new module:

- [ ] Test `plan` mode shows correct output
- [ ] Test `apply` mode works correctly
- [ ] Test idempotency (multiple runs)
- [ ] Test with module disabled in config
- [ ] Test error handling (missing dependencies, etc.)

### 3. Update Documentation

- [ ] Update `README.md` if adding user-facing features
- [ ] Update `.devboost.yaml.example` if adding config options
- [ ] Update `CHANGELOG.md` with your changes
- [ ] Update `AGENTS.md` if changing development guidelines

### 4. Commit Your Changes

Follow the [CBEAMS commit message style](AGENTS.md#4-commit-message-style-cbeams):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Examples:**

```
feat(zsh): add support for custom znap path

Allow users to configure znap installation path via
.zsh.znap_path in config file. Defaults to ~/.zsh-snap
if not specified.

Closes #42
```

```
fix(starship): correct git_status format syntax

The format string was using invalid variable concatenation.
Changed to use $all_status only, which is the correct
starship syntax.

Fixes #38
```

```
test(linux): add Ubuntu 22.04 testing

Verified package installation and module functionality
on Ubuntu 22.04. All modules working correctly.

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

Adding modules is designed to be super easy! See [AGENTS.md](AGENTS.md#3-module-development) for detailed instructions.

**Quick checklist:**

1. Create `modules/module_foo.sh`
2. Implement `db_module_foo_register()`, `plan()`, and `apply()`
3. Add to `build.sh` (file inclusion + registration)
4. Add config options to `.devboost.yaml.example`
5. Test thoroughly
6. Update documentation

## Code Quality Standards

- **Bash best practices**: See [AGENTS.md](AGENTS.md#2-code-quality-standards-2025)
- **Error handling**: Always check return codes, provide helpful messages
- **Performance**: Minimize external calls, cache when appropriate
- **Readability**: Clear function names, consistent naming, focused functions
- **Security**: Never execute user input, validate paths, backup before modify

## Testing Requirements

**All code must be tested before submission.**

### Minimum Testing Requirements

1. **Build and syntax**: `./build.sh && bash -n dist/devboost.sh`
2. **Plan mode**: `./dist/devboost.sh plan` (should not error)
3. **Apply mode**: `./dist/devboost.sh apply` (should work)
4. **Idempotency**: Run `apply` twice, second should be no-op

### Recommended Testing

- Test with different config files
- Test error conditions (missing dependencies, etc.)
- Test on different operating systems if possible
- Test edge cases

### Testing on Different Platforms

We especially welcome contributions that test and fix issues on:
- Different Linux distributions (Ubuntu, Debian, Fedora, Arch)
- Different macOS versions
- Different shell versions

If you test on a platform, please note it in your PR!

## Reporting Bugs

When reporting bugs, please include:

1. **Environment**:
   - OS and version
   - Shell version
   - devboost version

2. **Steps to reproduce**:
   - Exact commands run
   - Config file (if applicable)

3. **Expected behavior**:
   - What should happen

4. **Actual behavior**:
   - What actually happened
   - Error messages
   - Logs (with `--verbose` flag)

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
   - Documentation updates
   - Backwards compatibility
3. Be open to feedback and suggestions
4. Address review comments promptly

## Questions?

- Check [AGENTS.md](AGENTS.md) for development guidelines
- Check [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- Open an issue for questions or discussions
- Be respectful and constructive in all interactions

## Recognition

Contributors will be:
- Listed in the README (if desired)
- Credited in release notes
- Appreciated by the community! 🎉

Thank you for contributing to devboost!

