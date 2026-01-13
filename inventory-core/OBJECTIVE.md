# Qualitative Objective

## Semantic Functionality

**Stores** include inventory resources with **amounts**, [using user-core] **groups** priveliged (separately) to read, write, and administrate (read/write + write privelige information for the group itself) **amounts**, **history-tracking** for the **amounts** (nobody can edit/immutable by users, admins and amount-readers can view), and **statistic reporting** (choose **prediction models** + reports generated from those) + [using user-core] **groups** for the **statistic reporting**

**Amounts** are quantities of a particular **resource** 

**Resources** are named entities which can have tags (arbitrary maps) and are of a particular **type** or **unit class** (the usage is interchangeable; consider equivalent)

**Units** are convertable to and from others within the same **unit class**

**Unit classes** contain particular units which must have a complete set of defined unit-conversions AND unit classes define a default unit

**Statistic Reporting** uses some choice of **Prediction Model** (many implemented) to attempt to predict a future amount of a resource (the "base statistic") at many different future time-steps or some choice of other statistic (e.g. days until resupply given desired resupply threshold for a decrease-over-time, resupply-under-threshold resource)

Functional traits of **Stores**:

**Stores** provide **triggers** which condition on resource amounts and have **effects** (either notifications (configured in user preferences) or [using task-core] task-dispatch or [using workflows] workflow-dispatch, etc)

**Stores** can be added to by write-enabled users; writes are logged (append-only; auditable up to a min time prior to today for data storage constraints) and treated as "reports", viewable by the PredictionModels to understand consumption/production behavior for future prediction

All user data (e.g. preferences) is stored in the user-core's data store; user-core should handle authentication, identity, and access management (all AA handling).

## Usability and User-Focused Objectives

priorities: [1] is the highest, [5] is the lowest

A [1] blazing-fast (runtime), [1] hyper-convenient (for "fast-track" usages), [3] keyboard (vim-style)/shortcut (microsoft alt-chord style)-driven, and [1] data-first (i.e. [1]maximal information in screen-space; [3] configurable where necessary) dashboard provides users with:
- a clear view of a [1] resource amount "history graph" on screen A for resource X ([1]X is switchable; screen A remains in place)
- [3] a (multi-)selection version (screen B, perhaps mutated from screen A or perhaps separate) of screen A whcih shows for resource list X* graphs of them all, possibly paged but ideally showing many graphs in a page
- for every "history graph": a graph with proper [1] labelling + title, but also [2] summary statistics on the left such as time since last update ([1] live-updating), a minimum size for visibility when "minimzed" however an option to "maximize" i.e. fullscreen the graph
- a way to report inventory levels ([1] this is a "fast track" item which must be at least possible with the minimum required clicks/workflow complexity to the user -- clever [dynamic autofill ideally] defaults/"extension framing" for optional information to only be visibly considered after required information is all presented) i.e. a form ([3] with vim-style fast keyboard-based fillout and [1]very easy to use on mobile)
- [2] some kind of NFC tag integration to autofill elements of the form + pull it up to be placed at physical locations of stores to further hyper-convenience the user
- [3] notification settings configurable for this app configurable in its dashboard, as well as in [user-core]
- [2] clear error messages, especially access related ones
- [1] easy access on desktop, web, and [3] mobile (however just web is fine if the site is mobile-friendly)

# SLOs (Quantiative Contractual Objectives)
TODO: populate with API endpoints + throughput/benchmark targets + error guarantees
