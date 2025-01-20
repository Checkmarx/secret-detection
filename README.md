# AST CLI: Pre-Commit Secret Prevention

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

## Installation
1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd <repository-directory>
    ```

Install dependencies:
npm install
