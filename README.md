# Compell
The no fluff no bs coding assistant.
- No workflows and tools forced on you. Pick your own tools and workflows.
- Work with any LLM
- Configurable default tools which restrict the scope of the assistant's capabilities.
  - Hide files and directories from the assistant's view
  - Make files and directories read only for the assistant
  - Limit the os commands the assistant can execute
  - Modes
    - Prompt for approval and confirmation
    - Full autopilot
  - Session customization
    - Specify tools available in the session explicitly
    - Specify toolsets available in the session
      - Toolsets defined in agent configuration
- Work with any available MCP servers in addition to the default one used for local filesystem access and command execution
  - Specify in agent configuration what additional MCP servers to use
- Tools from additional MCP server to be specified in toolset definition and session level disable as <server name>:<tool name>
  - Default tools to be named as <tool name>
- Default tools
  - Read whole repo into context
  - Read file into context
  - Write to file
    - Replace whole file
    - Replace part of file
  - Create and delete empty directory
  - Delete file
  - Execute command
    - Only commands from a predefined list
      - Regex and wildcard expressions for commands to allow
- Agent configuration
  - Can be specified in user level in ~/.compell/config.yaml or at directory level in .compell/config.yaml
  - Example configuration:
    ```yaml
    toolsets:
      - name: read_only
        tools:
        - read_repo
        - read_file
        - get_current_time
      - name: default # Special toolset that gets loaded when no other toolset is specified
        tools:
        - read_repo
        - read_file
        - write_file
        - create_dir
        - delete_file
        - execute_command
        - get_current_time
        - google-pse-mcp:search
    additional_mcp_servers:
      - name: google-pse-mcp
        command: npx
        args:
         - -y
         - google-pse-mcp
         - https://www.googleapis.com/customsearch
         - <api_key>
         - <cx>
    allowed_commands:
      - whoami
      - 'git .*' ' # Allow all git commands via regex
    filesystem_access:
     hidden:
      - test_notes.md
      - '**/node_modules
     read_only:
      - plan.md
    ```
- Agent invocation
  ```
  compell -m auto/prompt -s <session_name> -t <toolset_name>
  ```
  - Session name defaults to <current directory name>_<execution timestamp in YYYY-MM-DD HH:mm:ss> if not specified
  - If toolset is not specified, default toolset is used
- Resume session
  ```
  compell -r <session_name>
  ```
- Session is saved in .compell/sessions/<session_name> in current directory
- The .compell directory is by default invisible to the agent
