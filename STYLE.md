# DESIGN
## "many stub"
Avoid the "many stub" problem -- when you have one system which originally interacts with a few but adding interactions with new services requires both work on the new service end AND the original service end 

For instance, consider the scenario:
Original service had a list of types which it instantiates and uses for instance as an EnemySpawner class, and writing a new enemy requires adding the enemy itself, and an entry to this list

Pub/sub triggers are an example solution to the system-design level version of this problem for notifications

## Core System Requirements
All systems should have logging, metrics, and events (pub/sub style triggers which are easy to plug and play)

For this project, we also require all systems to have gRPC endpoints and use Protocol Buffers (Protobuf/Proto definitions) to share schemas and interact with each others

# PROTO
Prefer `oneof` fields to enums when packing more than one piece of information into a 'switchable' construct; a telltale indicator of this should be when you need to make other fields act optional or required contingent on the value of an enum field.

Document all rpc methods and their request and response messages as a supreme priority; try to document everything else too, at a lower priority.

Prefer nesting over chains of related fields which share a prefix. 

Always reserve fields upon deletion/schema refactors **if and only if** code has already gone into prod; otherwise refactor away -- it improves cleanliness

# GO
Always prefer errors to panics unless continuing in an otherwise panic-able branch would produce undefined behavior with the property it isn't recoverable without creating damage to the system in some form which persists (on-disk, for instance) or passes forward incorrect/invalid information to another server without some kind of flag or indicator it is invalid. 

`nil` objects should ONLY ever be returned alongside an error as the last return value -- `nil` dereference exceptions should never occur as sany function that uses errors should also ONLY use output after checking for errors (use `[]string{}` over `nil` for a slice, similarly for a map)

Always wrap errors with some text when they come from a subcall and have the error message capture the thing the subcall did (which should be close to the function name in readable code), alongside any meta-manipulation not captured by that which happened inside the function. 

The "wrapping message" described above or the message returned for an error generated inside this function itself should almost never contain anything along the lines of the function name for the stated reasons. When wrapped according to the above we should get something like "name of subfunction failed (if minorly modified in a way like casting that's fine to mention): reason (scoped to provide information assuming it's inside the subfunction)"

Also use structured logging (logrus), and follow the philosophy of fields over natural language -- minimize the segment of an error a human would have to parse through gramatically if possible but make that segment at least 2-5 words depending on the complexity of the fields to provide enough context to understand all of the other fields associated with it instantly, without checking the error spec (see logs in home-manager subrepo as reference)

Never use `else` or `else if`. Switch statements or the conditional inversion pattern will be able to take care of the majority of the cases `else` may seem syntactically required; in others where it may seem impossible to functionally maintain code correctness while removing else, remember the quick-return pattern is possible, contingent on the size of the outer scope of the code in question. 

If it seems difficult to refactor out an `else`, your function is too big and fundamentally violates single-responsibility principle. Trim the scope and use subcalls with quick-returns.

Support the `if val, ok := useful_map[key] ; ok` pattern where possible, inverting the condition if necessary, even when resulting in cases like:
```go
if _, ok := useful_map[key] ; !ok {
    return nil, error
}
value := useful_map[key]
```
where it may seem redundant, this code ends up being more readable so it's worth the style determination

