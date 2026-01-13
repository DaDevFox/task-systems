All commit (first-line/oneline) messages follow the following format:
`ACTION(SCOPE): [MODIFIER]MESSAGE`

ACTION and SCOPE text always utilize 4 character abbreviations OR _ right padding to 4 characters and are uppercase

# ACTIONs

ACTIONs explain how the SCOPE base (system) is modified by this change, in terms of functional impact to other possible SCOPE bases which interface with this one

"Functional" here refers to modification of logic, i.e. a change modifying the provisioning of a thing which *does* something (rather than is something) -- this change spawns something which can be considered an autonomous executor in an isolated enough scope. Most code changes in a programming or scripting language are "functional", most code changes in a data or configuration language are "nonfunctional".

They can be any of:

## Nonfunctional Change Indicative ACTIONs
- `FMT_`: formatting -- no functional change + no substantive text added or changed -- just moved around; MESSAGE should start with noun (thing which is formatted)
- `CFG_`: configuration -- no functional change + text added or changed which is not documentation; MESSAGE should start with noun (thing which is configured)
- `DOC_`: documentation -- no functional change, but changes to documentation (internal or external); MESSAGE should start with noun (thing which is documented)
- `MERG`: merge -- denotes a merge commit; SCOPE in this specific case is XXXX and MESSAGE can start with blank or summary of changes (choose a commit message from merged changes)

## Functional Change Indicative ACTIONs

- `GEN_`: generation -- generation of functional code (heavily discouraged); MESSAGE should start with noun (thing which is generated)
- `ICM_`: initial commit -- first commit of a significant and unversioned corpus of functional code (discouraged); MESSAGE should start with noun (thing which is committed)
- `FIX_`: fix -- functionally changes something already described adequately in another commit as `REF_`, `IMP_`, `ENH_`, or `FEAT`; MESSAGE should start with verb (what this code does); see MODIFIER `b` to describe the bug which is fixed instead
- `REF_`: refactor -- prepares a functional change, but does not itself change active functionality (e.g. non-injected-into-main-control-flow code); MESSAGE should start with "to" followed by verb (explaining execution of functionality prepared for -- what that code eventually does)
- `IMP_`: improvement -- functional change towards subcomponent or satellite functionality to a SLO-affecting system; MESSAGE should start with verb (what this code does)
- `ENH_`: enhancement -- functional change which improves existing SLO-affecting capability of the SCOPE; MESSAGE should start with verb (what this code does)
- `FEAT`: feature -- functional change which adds a new SLO-affecting capability to the SCOPE; MESSAGE should start with verb (what this code does)

non-(release)-versioned ACTIONs:
ICM_ FMT_ CFG_ IMP_ REF_ DEL_ MERG DOC_

PATCH-(release)-versioned ACTIONs:
FIX_ ENH_

MINOR-(release)-versioned ACTIONs:
FEAT

# SCOPEs

every base SCOPE has a SLO (System Level Objective -- ideally documented) which is a defined set of functionality and performance characteristics. The SLO gives all information another SCOPE needs to know to interact with this one. Every SCOPE also is part of a category, called the base SCOPE which is essentially a "release target" -- the name of a product as it will be released; the other part of a SCOPE is called its subsystem. 

base SCOPEs in this project are generally going to relate to top-level directories of the repo, but currently:

- `ALL_`: all projects
- `INV_`: inventory-core
- `WKFL`: workflow-core (workflows)
- `TASK`: task-core (tasker-core)
- `SHRD`: shared

SCOPE subsystems are generally one of:
- `VCS_`: version control system (git or jj)
- `IDE_`: integrated development environment (vim/nvim, vscode, etc)
- `PROT`: proto configurations 
- `CICD`: CI/CD capabilities to ensure SLO functionality reliability
- `SRV_`: service/API
- `FRNT``: frontend/UI

any work towards a SCOPE SLO which doesn't fit into a subsystem is simply the base SCOPE and denoted as e.g. `ALL_`. SCOPEs with a subsystem are denoted as e.g. `INV_/VCS_`

any work towards the `ALL_` base SCOPE which has a subsystem can simply be denoted by the subystems e.g. `VCS_` instead of `ALL_/VCS_`

# MODIFIER

if change breaks a SLO MODIFIER is `!` (e.g. full message prefix: `FEAT(INV_/SRV_): [!]end v1 API support`)

if message should start with a noun and a verb is preffered, namely when describing what the **d**eveloper did rather than what the code does, the modifier is `d`

> NOTE: messages should never use a noun to start if a verb is preferred -- above case _adds_ infromation, this would reduce information which destroys readability (disambiguation is difficult)

for a message with ACTION `FIX_`:
if message should start with a verb describing new functionality and a noun, namely the bug which is fixed, is preferred the modifier `b` may be used and the noun may be described in this specific case, as the verb 'fix' is implied by the ACTION


