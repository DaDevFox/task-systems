# Qualitative Objective

## Semantic Functionality

**Task Domains** are groupings of **Tasks** with [from user-core] **groups** for reading/task completion/writing (creating + assigning) access

**Tasks** are assigned to [from user-core] **Users** and have a **state** (enum), which **Triggers** can watch for changes, **deadlines** they should be completed by, **Results** which are objects that are filled as part of the task and must be set before completion, and **dependencies** which must be satisfied before completion (these can be **Results** of other tasks (or whole other **Tasks**) or another thing called a **Resource**). 

**Tasks** may spawn subtasks, whose completion in entirety becomes a **Result** of the original (now parent) task.

**Tasks** originate from **TaskFactory**s which simply define a [from user-core]group which has the right to configure tasks and assign them to users, stored trivially (but may expand in future to more metadata) as a string tag in the **Task** to denote where they are from. 

**Users** can (and must) edit the **state** enum to completion to finish a task -- other systems can wait on this; but doing so will be blocked (and the task will be markedas blocked) if **dependencies** are not yet satisfied. 

**Results** may be files, form submissions, etc (usually have to do with proof of the work rather than the work itself but could be both). msut be awaitable.

**Resources** may be API calls, verification of group permissions, etc. Must be awaitable like a Task Result.

TODO: another service handles Resources (an automation serving system like n8n/NodeRED?)

**Tasks** can contain varieties of metadata which can be useful for analysis; typing/streaming of this data is also useful so **Tasks** can also implement **Traits**, e.g. `timeable` (tracks start + end time) or `pointed` (tracked for size/completion difficulty). **Traits**' data can be updated via custom programs (other services) sometimes but preferably acts through **Triggers** set to act on **Results** or state change

**Systems** act on **Tasks** of a particular **Trait** and (more informal; can really do anything with the trait info) seek to constrain **Task** execution rules, track stats (per-user or not), etc.

### Systems to implement:
Pomodoro using Timing trait with a separate Pomodoro desktop client (with configurable cycles/timings of course)

3-cycler system which enables the user (only) to cycle between tasks of 3 (different) topics to completion for a given time-duration after which the 3 topics may change

TODO: Other systems (I love Mark Foerster's systems such as Autofocus, adaptations of DIT, etc -- those would be great here)
## Usability and User-Focused Objectives

priorities: [1] is the highest, [5] is the lowest

A [1] blazing-fast (runtime), [1] hyper-convenient (for "fast-track" usages), [2] keyboard (vim-style)/shortcut (microsoft alt-chord style)-driven, and [2] data-first (i.e. [2]maximal information in screen-space, [2]configurable views) dashboard + task completion frontend provides users with:
- a clear table-view (screen A) of [1]tasks filtered by filter(s) X (filter by active status, user, etc)
- a clear hierarchy-view (screen B) of [1]tasks filtered by the same filter(s) X
- a clear timeline-view (screen C) of [1]tasks filtered by the same filter(s) X
- a clear DAG-view (screen D) showing task dependencies (shows more than just tasks -- this one adds visual representations of Resources, etc) of [1]tasks filtered by the same filter(s) X
  - A/B/C/D are a tab-switch at the top of a main screen
  - filters X can filter by active status, user, task domain, and (only) shared metadata
- ways (on screen A) to fulfill **Results** of any kind
- special screens of the form of A/B/C/D (and showing in the same place), extensibly defined by **Systems** implemented and filterable
- [1] clear error messages, especially access related ones
- easy access on [1] desktop, [2] web, and [4] mobile (however just web is fine if the site is mobile-friendly)


# SLOs (Quantiative Contractual Objectives)
TODO: populate with API endpoints + throughput/benchmark targets + error guarantees
