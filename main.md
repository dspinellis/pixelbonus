# Goal
In a single shot implement software according to the specifications
listed in the file `specs.md` or in the files `specs-*.md`.

# Method
- If the specifications are incomplete or not committed, stop with a message
  indicating what is missing.
- For large complex systems split the specifications into multiple
  small self-contained specification files (e.g., `specs-input.md`,
  `specs-processing.md`, `specs-output.md`)
  adding required interfaces.  Also add a file `spec-global.md`, which
  contains global functional and non-functional specifications applicable
  to all split parts.
  Stop and ask for the original specifications to be removed in favor
  of the split ones or for the original ones to
  be adjusted to specify a different split.
- If at some point you realize that the specifications are incomplete,
  stop and ask for help.  The help will entail adjusting the specifications.
- In a file named `progress.md` list each specification file, its
  short Git commit hash, and its status as `implemented` possibly
  followed by `tested`, and the associated code files.
- If specifications change the corresponding code files must be removed
  and recreated from the specifications ONLY.
- Create integration tests and run them once all specifications are
  implemented.
- Create a file named `implementation-notes.md` where you record
  any decisions made for implementing the code.
- Create a file named `README.md` with instructions on compiling,
  testing, deploying, running, and monitoring (if needed) the created
  software.

# Constraints
- Aim at code that can be automatically tested.
  - Split correspondingly the specifications.
  - Design and write testable code.
- Each specification shall be implemented separately and fully tested.
- Each specification shall only be implemented in terms of its
  specification file and `specs-global.md`. No other specification files
  shall be consulted.
- As much as possible add and run unit tests for all generated code.
- No need to create unit tests for trivial methods (e.g. getters and setters).
- The project must always be implementable from the specifications.

# Other notes
- Ignore editor swap files (e.g. .*.swp)
