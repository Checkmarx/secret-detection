## Overview
The **AST CLI** project introduces pre-commit secret scanning functionality to detect and prevent the exposure of sensitive information such as passwords, API keys, and tokens before they are committed to repositories. This tool empowers developers by enabling real-time secret detection as part of their Git workflows.

## Features
- **Secret Detection Module**: Scans for secrets during commit operations.
- **Pre-Commit Integration**: Automatically hooks into Git workflows using the `pre-commit` framework.
- **Ignore Management**: Supports ignoring specific findings via a `.checkmarx_ignore` file.
- **Command-Line Interface (CLI)**:
    - `install`: Sets up pre-commit hooks locally or globally.
    - `uninstall`: Removes pre-commit hooks.
    - `update`: Updates pre-commit hooks to the latest version.
    - `scan`: Executes a scan for secrets (internal use by hooks).
    - `ignore`: Adds specific findings to the ignore list.
- **License Validation**: Ensures only users with an active CxOne license can access the functionality.

# Contributing to Secret Detection

Thank you for considering contributing to the Secret Detection project! Here are some guidelines to help you get started.

## How to Contribute

1. **Fork the repository** and clone it to your local machine.
2. **Create a new branch** for your feature or bugfix.
3. **Make your changes** and commit them with clear and descriptive messages.
4. **Push your changes** to your forked repository.
5. **Create a pull request** to the main repository.

## Code of Conduct

Please adhere to our [Code of Conduct](CODE_OF_CONDUCT.md) in all interactions.

## Reporting Issues

If you find a bug or have a feature request, please create an issue using the appropriate template.

## Coding Standards

- Follow the existing code style and conventions.
- Write tests for new features and bug fixes.
- Ensure your code passes all tests and linter checks.

## Review Process

All contributions will be reviewed by project maintainers. Please be patient as we review your changes.

## Installation
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd <repository-directory>
    ```

Install dependencies:
npm install
