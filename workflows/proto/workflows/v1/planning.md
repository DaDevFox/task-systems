
# piles
cleaning (inc), need 0 
  subpiles: rooms; inherit behavior from above
laundry (inc), need 0 
dishes (inc), need 0
meals (negative inc, trigger raised), need positive
groceries (trigger raised from cooking), need 0

# task systems
## cleaning
clean on any subpile > threshold 
(trigger raised) ->
task assignee with clean action; wait for completion report
task reviewer with reviewing clean action; wait for completion report
(no end trigger)

## laundry
load on pile > threshold 
(trigger raised) -> (skip review)
task assignee with wash action; wait for completion report
(trigger raised after delay -- variable) -> (skip review)
task (same) assignee with dry action; wait for completion report
(trigger raised after delay -- variable) ->
task (same) assignee with unload action; wait for completion report
task reviewer with reviewing load action; wait for completion report
(trigger raised) -> (skip review)
task all with fold + hang action; wait for completion report
(no end trigger)

## dishes
wash on pile > threshold
(trigger raised) -> (skip review)
task assignee with wash, dry action; wait for completion report (field for dishwasher used)
(trigger raised after delay -- variable [dishwasher not used => 0])
task (new) assignee with unload action; wait for completion report

## food 
### meal planning
on schedule (e.g. during week sometime)
(trigger raised) -> (skip review)
[report/state mutation requested]: week's meal plan open for discussion
task assignee group (manpower need determined) with meal planning action; wait for completion report 
(no end trigger)

### grocery shopping
shop on meal pile < [same threshold as below] - 1 day OR on schedule (e.g. Saturday)
(trigger raised) -> (skip review)
task assignee group (manpower need determined) with grocery shopping action; wait for completion report
(no end trigger)

### cooking
cook on meal pile < threshold OR on schedule (e.g. Sunday)
(trigger raised) -> 
task assignee group (manpower need determined) with cook action
(no end trigger)




