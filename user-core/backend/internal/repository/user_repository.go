package repository

// Ensure repository implementations satisfy the core interface. Keeping this in a
// dedicated file avoids cyclic imports when wiring repositories from other
// packages (for example, via build tags or future persistence backends).

var (
	_ UserRepository = (*InMemoryUserRepository)(nil)
	_ UserRepository = (*BadgerUserRepository)(nil)
)
