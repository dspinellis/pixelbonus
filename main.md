# Goal
In a single shot implement software according to the specifications
listed in the file `specs.md` or in the files `specs-*.md`.

# Method
- If the specifications are incomplete or not committed, stop with a message
  indicating what is missing.
- If at some point you realize that the specifications are incomplete,
  stop and ask for help.  The help will entail adjusting the specifications.
- For large complex systems follow the instructions under the heading
  modular development.
- In a file named `progress.md` list each specification file, its
  short Git commit hash, and its status as `implemented` possibly
  followed by `tested`, and the associated code files.
- If the system uses a database add a file named `schema.sql` detailing
  the database's schema via SQL Data Definition Language statements
  and suitable comments.  Have the software initialize the database
  by means of that file.
- If specifications change the corresponding code files must be removed
  and recreated from the specifications ONLY.
- Create integration tests and run them once all specifications are
  implemented.
- Create a file named `implementation-notes.md` where you record
  any decisions made for implementing the code.
- Create a file named `README.md` with instructions on compiling,
  testing, deploying, running, and monitoring (if needed) the created
  software.

# Modular development
For large complex systems follow the instructions below.

- Select the most appropriate architecture for the specific
  system and document it in a file named `specs-architecture.md`.
- Split accordingly the specifications into multiple small
  self-contained specification files for separate implementation
  module (e.g., `specs-input.md`, `specs-processing.md`,
  `specs-output.md`) with clearly-defined interfaces.
- Add a file `specs-api.md` that specifies the API provided by each
  file implementing a specific part of the specifications.  This must
  provide the complete information needed to utilize that part of the
  specifications, namely what to import/link/include, each provided
  function/method/class/endpoint, its arguments, and return value.
- Add a file `specs-integration.md` specifying how the separately
  specified modules work together. This includes where any entry
  point resides and how the modules work together.
- Add a file `specs-global.md`, which contains other global functional and
  non-functional specifications applicable to all split parts.
- Create a file name `use-cases.md` where you name each use case
  and the associated specification file(s).
- Create a file `design-validation.md` where you describe the API
  calls involved for use case. If a use case cannot be implemented
  using the available APIs revise the associated APIs and specifications
  and repeat the process.
- After finishing the split design, stop and ask for the original
  specifications to be removed in favor of the split ones or for the
  original ones to be adjusted to specify a different setup.

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
- Do not repeat content from this file in other specification files.
- In split systems, do not not provide or use any  functionality from another
  module without using a corresponding API.
- Test all provided APIs.

# Other notes
- Ignore editor swap files (e.g. .*.swp)
