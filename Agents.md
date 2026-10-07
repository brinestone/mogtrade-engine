# MogTrade Engine

## Project Description
This project implements a trading platform's engine called MogTrade. It exposes a RESTful API for frontends and clients to connect to and permit users place trade orders on the connected stock market exchanges.

## Project Structure
The project is divided into golang modules under the following directories
- **`app`**: Contains code for running the main entrypoint executable, which uses an IoC Container and GIN framework to host the API server.
- **`core`**: Contains core library interfaces which can be used everywhere else.
- **`infra`**: This library module contains code for integrating external systems, in other words, implementation details of `core` abstractions for communicating with external systems.
- **`services`**: This module contains service functions for business core logic.

The project also contains the `go.work` and `go.sum` files which link the different modules together.

## Development Tools
This project uses several popular development tools like
- **`sqlc`**: For generating domain data models and query functions
- **`smithy`**: For creating and managing REST api documentation.
- **`air`**: For running local development application. It's description can be found in the .air.toml
- **`mcp server`**: Use the `mogtrade-dev-mcp` MCP server (located at `.vscode/mcp.json` or `.mcp.json`) for database migration file generation.

## Build and test commands
Use the standard golang commands for running and testing

## Code style
- Never use `interface{}` type, but the `any` type where needed.
- Use the `new(any)` built-in function that returns a pointer to the value passed, to obtain a pointer to a value.

## Ignored files
The following files/directories should not be accessed at all costs
- **`.vfox`**: This folder contains sdk symlinks managed by the Version fox sdk manager.
- **`app/cmd/.env`**: This file contains secret values which should not be exposed. For this, all agents are therefore forbidden from looking into this file. However, an example of this file also exists in the same directory: (**`app/cmd/.env.example`**) for variable reference names.
- Also ignore all files listed in the .gitignore files

## Conventions
- Generated identifiers are ULID values.
- Use the `contract.IdGeneratorFunc` function for creating IDs

### Branch Naming Convention
Always create a new git branch for each feature or bugfix before beginning work. Use descriptive branch names following the convention: `feat/<feature-name>`, `fix/<bug-description>`, or `refactor/<refactor-purpose>`. Never commit directly to the `main` or `master` branch. Open a Pull Request against `main` after completing the feature/bugfix and ensure all tests pass.

### Branch Creation Workflow
## Commit Message Convention
Use conventional commit messages following the format:
`type(scope): description`

Where `type` is one of:
- `feat`: A new feature
- `fix`: A bug fix
- `refactor`: A code refactor
- `docs`: Documentation changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `test`: Adding missing tests
- `chore`: Routine maintenance tasks

The `scope` is optional and should denote the area of the codebase (e.g., `auth`, `api`, `infra`).

Example commit messages:
- `feat(auth): add JWT authentication middleware`
- `fix(api): resolve rate limiting issue`
- `refactor(core): improve user query performance`
- `docs(readme): update contribution guidelines`

Ensure each commit addresses a single concern and is logically separate from other changes.
1. Create a new branch: `git checkout -b feat/your-feature-name`
2. Implement the feature/bugfix
3. Commit changes with descriptive messages
4. Push the branch: `git push origin feat/your-feature-name`
5. Open a Pull Request against `main`
6. Address code review feedback
7. Merge Pull Request after approval