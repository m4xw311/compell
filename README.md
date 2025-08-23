# Compell
The no fluff no bs coding assistant.

## Installation

To install Compell, you need to have Go installed (version 1.20 or later).

1. Install the binary:
   ```bash
   go install github.com/m4xw311/compell@latest
   ```
   This will install the `compell` executable in your `$GOPATH/bin` directory.

Alternatively, you can build from source:

1. Clone the repository:
   ```bash
   git clone https://github.com/m4xw311/compell.git
   cd compell
   ```
2. Build the binary:
   ```bash
   make build
   ```
   This will create an executable `compell` in the `bin/` directory.

## Usage

You can run Compell directly from the `bin/` directory, or if installed with `go install`, directly from your path:

```bash
compell [command-line-arguments] [initial-prompt]
```

**Example:**

To start a new session in prompt mode:
```bash
compell -m prompt "Refactor the 'agent' package to improve modularity."
```

To resume an existing session:
```bash
compell -r my_session_name
```

## Command Line Arguments

Compell accepts the following command-line arguments:

*   `-m`, `--mode` (string): Sets the execution mode.
    *   `prompt`: Compell will ask for user confirmation before executing any actions. (Default)
    *   `auto`: Compell will automatically execute actions without user intervention.
*   `-s`, `--session` (string): Specifies a name for the current session. If a session with this name doesn't exist, a new one will be created. If left empty, a default session name based on the current directory and timestamp will be used.
*   `-t`, `--toolset` (string): Defines which set of tools Compell should use for the session. Defaults to the `default` toolset defined in the configuration.
*   `-r`, `--resume` (string): Resumes a previously saved session by its name. When this flag is used, the `-s` flag is ignored if both are provided.
*   `--tool-verbosity` (string): Controls the verbosity of tool output.
    *   `none`: No tool output is shown. (Default)
    *   `info`: Displays basic information about tool execution.
    *   `all`: Shows all output from tool executions.

## Configuration

Compell loads its configuration from `config.yaml` files. It first looks for a user-level configuration at `~/.compell/config.yaml`, and then for a project-level configuration at `./.compell/config.yaml`. The project-level configuration overrides any conflicting settings in the user-level configuration.

An example `config.yaml` might look like this:

```yaml
llm: gemini
model: gemini-pro
toolsets:
  - name: default
    tools:
      - read_file
      - write_file
      - execute_command
      - read_dir
additional_mcp_servers:
  - name: my_custom_tool
    command: python
    args: ["/path/to/my_tool.py"]
allowed_commands:
  - git
  - go
filesystem_access:
  hidden:
    - .git/
    - node_modules/
  read_only:
    - important_docs/
```

### Configuration Options:

*   `llm` (string): Specifies the Large Language Model (LLM) client to use. Currently supported:
    *   `gemini`
    *   `mock` (for testing purposes)
*   `model` (string): Defines the specific model to be used by the chosen LLM client (e.g., `gemini-pro`).
*   `toolsets` (list of objects): A collection of toolset definitions. Each toolset object has:
    *   `name` (string): A unique name for the toolset (e.g., `default`, `python_dev`).
    *   `tools` (list of strings): A list of tool names that belong to this toolset.
*   `additional_mcp_servers` (list of objects): Allows you to define custom Multi-Client Protocol (MCP) servers. Each object includes:
    *   `name` (string): The name of the custom tool.
    *   `command` (string): The executable command for the tool.
    *   `args` (list of strings): Command-line arguments to pass to the tool.
*   `allowed_commands` (list of strings): A whitelist of shell commands that the agent is permitted to execute. If a command is not in this list, the agent will not be able to run it.
*   `filesystem_access` (object): Configures the agent's access to the filesystem.
    *   `hidden` (list of strings): A list of glob patterns for files and directories that the agent should not be able to see or interact with. The `.compell` directory is hidden by default.
    *   `read_only` (list of strings): A list of glob patterns for files and directories that the agent can read but not modify or delete.