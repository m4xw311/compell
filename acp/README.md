# ACP Package

This package implements the Agent Client Protocol (ACP) support for Compell.

## Overview

The Agent Client Protocol standardizes communication between code editors (IDEs, text-editors, etc.) and coding agents (programs that use generative AI to autonomously modify code).

Agents that implement ACP work with any compatible editor. Editors that support ACP gain access to the entire ecosystem of ACP-compatible agents.

## Usage

To enable ACP mode, use the `--acp` flag when running Compell:

```bash
compell --acp
```

In ACP mode, Compell communicates using JSON-RPC over stdio instead of the regular CLI interaction.

## Supported Methods

- `initialize` - Initializes the agent and returns capabilities
- `session/new` - Creates a new session
- `session/prompt` - Processes a prompt and returns the result

## Notifications

- `session/update` - Streams agent responses back to the client with `agent_message_chunk` updates