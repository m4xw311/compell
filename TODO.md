# Compell TODO List

## Agent Client Protocol
1. [ ] Implement the Agent Client Protocol support so that we can integrate with Zed
 - https://agentclientprotocol.com/overview/introduction

## 🛠️ Tool Implementation

2. [ ] **Default Tools:**
    2.2. [ ] **Limit Filesystem Access**: Make sure that the filesystem tools do not allow any operations outside the current working directory and it's subdirectories. Paths with .. and absolute paths outside the current directory should be denied.
3. [ ] **Session History**
    3.1. [ ] **Timestamp**: Add a timestamp to each session entry for later reference. Do not send this information to the LLM.

## ⚙️ Configuration & Usability

4.  [ ] **Graceful MCP Initialization Failure**: In `tools/tools.go`, if an MCP server fails to start, the error is just printed to the console. The application should handle this more gracefully, perhaps by warning the user that a specific toolset or tool is unavailable but allowing the agent to continue. - See if this is still the case.

## 📚 Documentation

5.  [ ] Add a section to the `README.md` on how to contribute, including instructions for setting up the development environment.
6.  [ ] Document the overall architecture of the agent, LLM client, and tool interaction flow.
7.  [ ] Add examples of how to create and register a new built-in tool.
8.  [ ] Add a guide on how to create and use a new MCP server for external tools.

## 🧪 Testing

9.  [ ] **Unit Tests for Core Packages**: Add comprehensive unit tests for key packages.
    9.1. [ ] `tools`: Test each built-in tool's `Execute` method with valid and invalid arguments.
    9.2. [ ] `config`: Test `LoadConfig` with different file setups (user-level, project-level, both). Test `GetToolset`.
    9.3. [ ] `session`: Test `New`, `Load`, and `Save` session functionality.
10. [ ] **Agent Integration Test**: Create an end-to-end test for the agent loop.
    10.1. [ ] Use a mock LLM client to simulate a conversation involving tool calls.
    10.2. [ ] Verify that the agent correctly parses tool calls, executes them, and sends the results back to the LLM.
    10.3. [ ] Test both `prompt` and `auto` modes.
