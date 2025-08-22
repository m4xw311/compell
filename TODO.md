# Compell TODO List

This document outlines the features and improvements planned for Compell, based on the `README.md` and comments in the source code.

## 🚀 High Priority Features

1.  [ ] **Implement Full Tool Call Execution Loop:** The core agent loop in `agent/agent.go` currently doesn't execute tool calls from the LLM. This is the most critical missing piece.
    1.1. [ ] In `agent/agent.go`, parse the `ToolCalls` from the LLM response.
    1.2. [ ] Implement logic to execute the requested tool with the provided arguments.
    1.3. [ ] Add support for `prompt` mode to ask for user confirmation before executing a tool call. `auto` mode should execute without prompting.
    1.4. [ ] Send the tool's output back to the LLM in a new message with `role: "tool"`, as noted in `llm/gemini.go`.
    1.5. [ ] Un-comment and implement the `ToolCalls` field in the `session.Message` struct in `session/session.go` to properly save and resume sessions with tool interactions.

## 🛠️ Tool Implementation

2.  [ ] **Implement Missing Default Tools:** Several standard filesystem and repository tools mentioned in the `README.md` and `tools/tools.go` are not yet implemented.
    2.1. [ ] **`read_repo`**: A tool to read the entire repository's file structure and content into the context. This is commented out in the default `config.yaml`.
    2.2. [ ] **`create_dir`**: A tool to create a new directory.
    2.3. [ ] **`delete_file`**: A tool to delete a file.
    2.4. [ ] **`delete_dir`**: A tool to delete an empty directory.
    2.5. [ ] **Enhance `write_file`**: Add functionality to replace a specific part of a file (e.g., by line numbers or a search/replace pattern), not just overwrite the whole file as the `README.md` suggests.

## ⚙️ Configuration & Usability

3.  [ ] **Update MCP Tool Naming in `README.md`**: The `README.md` specifies tool names as `<server name>:<tool name>`, but the code in `tools/tools.go` uses `<server name>.<tool name>`. The comment in `tools/mcp/mcp_tool.go` notes this was changed to fix an issue with the Gemini API. The documentation should be updated to reflect the current implementation.
4.  [ ] **Graceful MCP Initialization Failure**: In `tools/tools.go`, if an MCP server fails to start, the error is just printed to the console. The application should handle this more gracefully, perhaps by warning the user that a specific toolset or tool is unavailable but allowing the agent to continue.

## 📚 Documentation

5.  [ ] Add a section to the `README.md` on how to contribute, including instructions for setting up the development environment.
6.  [ ] Document the overall architecture of the agent, LLM client, and tool interaction flow.
7.  [ ] Add examples of how to create and register a new built-in tool.
8.  [ ] Add a guide on how to create and use a new MCP server for external tools.