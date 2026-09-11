# MogTrade Engine

## Project Description
This project implements a trading platform's enging called MogTrade. It exposes a RESTful API for frontends and clients to connect to and permit users place trade orders on the connected stock market exchanges.

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
- **`golang-migrate`**: For database migration in postgres
- **`smithy`**: For creating and managing REST api documentation.
- **`air`**: For running local development application. It's description can be found in the .air.toml

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