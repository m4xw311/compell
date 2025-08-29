# Introduction

The Agent Client Protocol standardizes communication between code editors (IDEs, text-editors, etc.) and coding agents (programs that use generative AI to autonomously modify code).

Agents that implement ACP work with any compatible editor. Editors that support ACP gain access to the entire ecosystem of ACP-compatible agents. This decoupling allows both sides to innovate independently while giving developers the freedom to choose the best tools for their workflow.

# Overview

ACP assumes that the user is primarily in their editor, and wants to reach out and use agents to assist them with specific tasks.

Agents run as sub-processes of the code editor, and communicate using JSON-RPC over stdio. The protocol re-uses the JSON representations used in MCP where possible, but includes custom types for useful agentic coding UX elements, like displaying diffs.

The default format for user-readable text is Markdown, which allows enough flexibility to represent rich formatting without requiring that the code editor is capable of rendering HTML.

# Architecture

The Agent Client Protocol defines a standard interface for communication between AI agents and client applications. The architecture is designed to be flexible, extensible, and platform-agnostic.

## Design Philosophy

The protocol architecture follows several key principles:
1. MCP-friendly: The protocol is built on JSON-RPC, and re-uses MCP types where possible so that integrators don't need to build yet-another representation for common data types.
2. UX-first: It is designed to solve the UX challenges of interacting with AI agents; ensuring there's enough flexibility to render clearly the agents intent, but is no more abstract than it needs to be.
3. Trusted: ACP works when you're using a code editor to talk to a model you trust. You still have controls over the agent's tool calls, but the code editor gives the agent access to local files and MCP servers.

## Setup

When the user tries to connect to an agent, the editor boots the agent sub-process on demand, and all communication happens over stdin/stdout.

Each connection can suppport several concurrent sessions, so you can have multiple trains of thought going on at once.

ACP makes heavy use of JSON-RPC notifications to allow the agent to stream updates to the UI in real-time. It also uses JSON-RPC's bidrectional requests to allow the agent to make requests of the code editor: for example to request permissions for a tool call.

## MCP

Commonly the code editor will have user-configured MCP servers. When forwarding the prompt from the user, it passes configuration for these to the agent. This allows the agent to connect directly to the MCP server.

The code editor may itself also wish to export MCP based tools. Instead of trying to run MCP and ACP on the same socket, the code editor can provide its own MCP server as configuration. As agents may only support MCP over stdio, the code editor can provide a small proxy that tunnels requests back to itself:

# Protocol
Protocol documentation links
- https://agentclientprotocol.com/protocol/overview.md
- https://agentclientprotocol.com/protocol/initialization.md
- https://agentclientprotocol.com/protocol/session-setup.md
- https://agentclientprotocol.com/protocol/prompt-turn.md
- https://agentclientprotocol.com/protocol/content.md
- https://agentclientprotocol.com/protocol/tool-calls.md
- https://agentclientprotocol.com/protocol/file-system.md
- https://agentclientprotocol.com/protocol/agent-plan.md
- https://agentclientprotocol.com/protocol/schema.md

JSON Schema:
- https://github.com/zed-industries/agent-client-protocol/blob/main/schema/schema.json

Example agent supporting ACP in Rust:
- https://github.com/zed-industries/agent-client-protocol/blob/main/rust/example_agent.rs
