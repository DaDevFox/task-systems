# INSTRUCTIONS

Look at the STYLES.md file to ensure code is readable.

Try to look at individual commits and give suggestions on how to split things up so that every individual commit is close to single-responsibility ("close" meaning it's fine to ignore this if commits are below a certain size, which I'll say here is 100 lines).
That is, explicitly label which lines could be in what new separate commits potentially.

Keep in mind these smaller commits should be buildable (not necessarily passing tests individualy, but must pass test by the end of the branch as far as the PR is aware) except when one explicitly mentions another as an extension of that original incomplete commit.
