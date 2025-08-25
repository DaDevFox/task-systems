# Contributing to the Monorepo

First off, thank you for considering contributing to this project! Your help is greatly appreciated. This document provides guidelines to help you get started.

## Table of Contents
- [Contributing to the Monorepo](#contributing-to-the-monorepo)
  - [Table of Contents](#table-of-contents)
  - [Commit Message Guidelines](#commit-message-guidelines)
    - [Commit Message Format](#commit-message-format)
    - [Semantic Versioning](#semantic-versioning)
  - [Adding a New Project](#adding-a-new-project)
  - [Coding Style](#coding-style)

## Commit Message Guidelines

We use a structured commit message format to enable automated semantic versioning and to generate clear, informative changelogs. Following these conventions is essential.

### Commit Message Format

Each commit message consists of a **header**, a **body**, and a **footer**. The header has a special format that includes an **ACTION**, a **SCOPE**, **MODIFIER** (enclosed by brackets if present, but optional), and a **subject**:

```
ACTION(SCOPE): [MODIFIER]MESSAGE
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

**The header is mandatory** and the scope of the header is the name of the project directory you are working on.

**Type**: This describes the kind of change you are making. See `SCOPES.md` for all possible types; some of which follow:
*   **FEAT**: A new feature. (Results in a `minor` version bump)
*   **FIX_**: A bug fix. (Results in a `patch` version bump)
*   **ENH_**: An enhancement to an existing feature. (Results in a `patch` version bump)
*   **IMP_**: A non-prod/non output-affecting change which prepares an enhancement. 
*   **DOC_**: Documentation only changes.
*   **PERF**: A code change that improves performance.
*   **TEST**: Adding missing tests or correcting existing tests.

**Scope**: The scope specifies the project affected by the change. This should be the `scope` value from the project's `.project.yml` file (e.g., `user-core-be`, `inventory-fe`, etc.) and should have an associated 4-character string used for commit messages, e.g. `USER/SRV_` or `INV_/FRNT`.

**Subject**: The subject contains a succinct description of the change:
*   Use the imperative, present tense: "run" not "ran" nor "runs".
*   Describe what the resulting code does (functional impact), not what you did (development change). See MODIFIER cases in SCOPES.md for exceptions to this rule.
*   Don't capitalize the first letter.
*   No period at the end.

**Breaking Changes**: To indicate a breaking change, use the `!` MODIFIER. This will result in a `major` version bump.
Example: `FEAT(USER/SRV_): [!]use <name of model> authentication model`

For more details, please refer to the `SCOPES.md` file at the root of the repository.

### Semantic Versioning

The commit message types directly influence the semantic versioning of each project, which follows the format `vSUPERMAJOR.MAJOR.MINOR.PATCH`.

When changes are pushed to the `main` branch:
- A **MAJOR** version bump (`v1.2.3.d` -> `v2.0.0.a`) is triggered by:
  - A commit with a `!` modifier (e.g., `FEAT(SCOPE): [!]...`).
  - A manual trigger of the workflow with the "Force a MAJOR version bump" option.
  - The closure of a GitHub Milestone.
- A **MINOR** version bump (`v1.2.3.d` -> `v1.3.0.a`) is triggered by:
  - A commit with the `FEAT` or `FIX_` type.
- An alphabetical **PATCH** version bump (`v1.2.3.a` -> `v1.2.3.b`) is triggered by:
  - A commit with the `ENH_` type.

Commits with types like `IMP_`, `DOC_`, `TEST`, etc., do not affect the version number. This automated system depends entirely on correctly formatted commit messages.

## Adding a New Project

When you add a new component (e.g., a new service backend or frontend) that needs to be versioned and released independently, you must make it discoverable by our CI/CD system.

1.  **Create a `.project.yml` file** in the root directory of your new project. This file contains metadata for the CI system. Here is a template:
    ```yaml
    # The scope used in commit messages, e.g., my-new-app-be
    scope: my-new-app-be 
    # The name used for releases and Docker images, e.g., my-new-app-backend
    release-name: my-new-app-backend
    # The path to the project's Dockerfile
    dockerfile: my-new-app/backend/Dockerfile
    # The type of project
    type: go-backend # or csharp-frontend, web-frontend
    ```

2.  **Update the Manual Release Options**: To make your new project appear in the manual release dropdown on GitHub, you need to update the CI workflow file. We have a script to do this for you.

    Run the appropriate command for your system from the repository root:
    - For Linux/macOS (or Git Bash/WSL on Windows):
    ```bash
    bash ./.github/workflows/update-projects.sh
    ```
    - For Windows (using PowerShell):
    ```powershell
    ./.github/workflows/update-projects.ps1
    ```
    This will find your new project and automatically add it to the list in `.github/workflows/comprehensive-ci.yml`.

3.  **Commit the change** to `.github/workflows/comprehensive-ci.yml`.

## Coding Style

All code should be well-formatted, readable, and consistent. Please refer to the `STYLE.md` file for detailed coding style guidelines for each language used in this repository.
