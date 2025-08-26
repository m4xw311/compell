3. [ ] Web interface - Rethink this - No need, seems like a bad idea
    3.1 [ ] The current purely CLI based user input is very limiting. It would be helpful if we have access to a web interface where the user interaction with the agent is in a more user friendly interface.
       3.1.1 [ ] Current cli functionality should work as is for situations where a browser is not available
       3.1.2 [ ] Additional cli argument `--web` to enable the web interface
          3.1.2.1 [ ] Basic chat interface in UI implemented with some proper UI framework - use out of the box components
          3.1.2.2 [ ] Ability for user to view chat history and resume a chat session
          3.1.2.3 [ ] Ability for user to start a new session
          3.1.2.4 [ ] Helper widget in the interface to speed up the user interaction. For example, a file selection widget to auto-complete file paths.
    3.2 [ ] Implement the server backend for the web interface in Go
    3.3 [ ] Implement the client frontend for the web interface using some prebuilt components
