# Qualitative Objective

## Semantic Functionality

Key functional trait of a **User**:

**Users** are entities which can be authenticated via a system exposed to the "outside" (external to the domain of "protected operation" all other services in this monorepo operate within). (Only) once authenticated, any and all access to services and data within the domain of protected operation may occur. 

**Users** store settings and miscellaneous metadata (called their "baggage") both pertaining to general processes and specificity of services that use this user-core service, where data residency is managed securely within the protected operation domain and further by this user service specifically (neither anyone [external or internal] who is not specifically authenticated user A, nor other services -- even acting on behalf of authenticated user B, C, ... can get any baggage data about the user; and authenticated user A relies on the user-core connection to get it at all). Other servies can request user baggage data iff they have the user for whom baggage is requested authenticated. 

**Users** are subject to a system of identity and access management that manages authorization for the entirety of the application system. Users can belong to **groups** as **owner**, **admins**, or (just) **members** (all 3 are "members", owners and admins have that + membership though), and any authenticated service within the protected operation domain (service acting on behalf of an authenticated user) can query whether a given user is a **member** of a given **group**. 

**Admins** may add/remove members to groups excepting **owners** and may not set others to **admin**. All admins are also members.

**Owners** may set members to be admin or transfer ownership to another. All owners are also members.

**Groups** can "subsume" (absorb) other groups (act as users and nest under them): when group A subsumes group B, it means every member of group B is part of group A or group B is a subset of group A (think: A subsumed i.e. absorbed i.e "included" group B, extending its membership by including those in B)

TODO: consider conditional subsumption

## Usability and User-Focused Objectives

priorities: [1] is the highest, [5] is the lowest

A [1] blazing-fast (runtime), [3] hyper-convenient (for "fast-track" usages), [2] keyboard (vim-style)/shortcut (microsoft alt-chord style)-driven, and [4] data-first (i.e. [3]maximal information in screen-space; [4] configurable where necessary) dashboard provides users with:
- a clear view (screen A) of [1]users in a group(s) X (X is switchable, multi-selectable + option for intersect versus union with multiple groups)
- ways (on screen A) to add/remove members and permissions, as per ownership/admin/member privelage restrictions
- [1] clear error messages, especially access related ones
- easy access on [1] desktop, [3] web, and [5] mobile (however just web is fine if the site is mobile-friendly)
- [1] a "settings" menu which shows all baggage with hierarchical source information (showing what services they relate to) and full editability + [3] a packagable component other services can implement to show the same menu (frontend; core data/backend info is sliced by what that service can access from user-core i.e. that which relates to it *pending policy change??) in those other client services with the same styling + format for consistency :D
- [3] a hierarchical view of group subsumption structures, with colored lines showing whether relations are admin/owner/user privilages or bolded for the user themself
- easy access on [1] desktop, [5] web, and [5] mobile (however just web is fine if the site is mobile-friendly)


# SLOs (Quantiative Contractual Objectives)
TODO: populate with API endpoints + throughput/benchmark targets + error guarantees
