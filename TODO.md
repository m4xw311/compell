# Compell TODO List

## 🛠️ Tool Implementation

1. [ ] **Default Tools:**
    1.2. [ ] **Limit Filesystem Access**: Make sure that the filesystem tools do not allow any operations outside the current working directory and it's subdirectories. Paths with .. and absolute paths outside the current directory should be denied.
2. [ ] **Session History**
    2.1. [ ] **Timestamp**: Add a timestamp to each session entry for later reference. Do not send this information to the LLM.
3. [ ] Web interface - Rethink this
    3.1 [ ] The current purely CLI based user input is very limiting. It would be helpful if we have access to a web interface where the user interaction with the agent is in a more user friendly interface.
       3.1.1 [ ] Current cli functionality should work as is for situations where a browser is not available
       3.1.2 [ ] Additional cli argument `--web` to enable the web interface
          3.1.2.1 [ ] Basic chat interface in UI implemented with some proper UI framework - use out of the box components
          3.1.2.2 [ ] Ability for user to view chat history and resume a chat session
          3.1.2.3 [ ] Ability for user to start a new session
          3.1.2.4 [ ] Helper widget in the interface to speed up the user interaction. For example, a file selection widget to auto-complete file paths.
    3.2 [ ] Implement the server backend for the web interface in Go
    3.3 [ ] Implement the client frontend for the web interface using some prebuilt components
## ⚙️ Configuration & Usability

2.  [ ] **Graceful MCP Initialization Failure**: In `tools/tools.go`, if an MCP server fails to start, the error is just printed to the console. The application should handle this more gracefully, perhaps by warning the user that a specific toolset or tool is unavailable but allowing the agent to continue. - See if this is still the case.

## 📚 Documentation

3.  [ ] Add a section to the `README.md` on how to contribute, including instructions for setting up the development environment.
4.  [ ] Document the overall architecture of the agent, LLM client, and tool interaction flow.
5.  [ ] Add examples of how to create and register a new built-in tool.
6.  [ ] Add a guide on how to create and use a new MCP server for external tools.

## 🧪 Testing

7.  [ ] **Unit Tests for Core Packages**: Add comprehensive unit tests for key packages.
    7.1. [ ] `tools`: Test each built-in tool's `Execute` method with valid and invalid arguments.
    7.2. [ ] `config`: Test `LoadConfig` with different file setups (user-level, project-level, both). Test `GetToolset`.
    7.3. [ ] `session`: Test `New`, `Load`, and `Save` session functionality.
8. [ ] **Agent Integration Test**: Create an end-to-end test for the agent loop.
    8.1. [ ] Use a mock LLM client to simulate a conversation involving tool calls.
    8.2. [ ] Verify that the agent correctly parses tool calls, executes them, and sends the results back to the LLM.
    8.3. [ ] Test both `prompt` and `auto` modes.
