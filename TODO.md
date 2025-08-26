# Compell TODO List

## 🛠️ Tool Implementation

1. [ ] **Default Tools:**
    1.2. [ ] **Limit Filesystem Access**: Make sure that the filesystem tools do not allow any operations outside the current working directory and it's subdirectories. Paths with .. and absolute paths outside the current directory should be denied.
2. [ ] **Session History**
    2.1. [ ] **Timestamp**: Add a timestamp to each session entry for later reference. Do not send this information to the LLM.

## ⚙️ Configuration & Usability

3.  [ ] **Graceful MCP Initialization Failure**: In `tools/tools.go`, if an MCP server fails to start, the error is just printed to the console. The application should handle this more gracefully, perhaps by warning the user that a specific toolset or tool is unavailable but allowing the agent to continue. - See if this is still the case.

## 📚 Documentation

4.  [ ] Add a section to the `README.md` on how to contribute, including instructions for setting up the development environment.
5.  [ ] Document the overall architecture of the agent, LLM client, and tool interaction flow.
6.  [ ] Add examples of how to create and register a new built-in tool.
7.  [ ] Add a guide on how to create and use a new MCP server for external tools.

## 🧪 Testing

8.  [ ] **Unit Tests for Core Packages**: Add comprehensive unit tests for key packages.
    8.1. [ ] `tools`: Test each built-in tool's `Execute` method with valid and invalid arguments.
    8.2. [ ] `config`: Test `LoadConfig` with different file setups (user-level, project-level, both). Test `GetToolset`.
    8.3. [ ] `session`: Test `New`, `Load`, and `Save` session functionality.
9. [ ] **Agent Integration Test**: Create an end-to-end test for the agent loop.
    9.1. [ ] Use a mock LLM client to simulate a conversation involving tool calls.
    9.2. [ ] Verify that the agent correctly parses tool calls, executes them, and sends the results back to the LLM.
    9.3. [ ] Test both `prompt` and `auto` modes.
