/*
	Repeatr is focused on telling a story about formulas: when you put the same
	things in, you should get the same things out.

	Formulas describe a piece of computation, its inputs, and how to collect
	its outputs.  After that, repeatr can help you make sure your formula
	produces the same thing time and time again.

	We call the parts of the formula that should be deterministic the "conjecture".
	We'll use this word consistently throughout the documentation.
	Anything that is part of the conjecture is hashed when processing the formula,
	and any output marked as part of the conjecture is hashed after the formula's task
	is executed.  (You can choose which of the outputs are part of your conjecture!
	But everything about your inputs must be part of your conjecture, because
	if the inputs change, output consistency is impossible -- except stuff like
	the network locations of data is skipped from the conjecture, since
	that can change without changing the meaning of your formula.)

	### Mathwise:

	Given a Formula j, and the []Output v, and some hash h:

	h(j.Inputs||j.Accents||filter(j.Outputs, where Conjecture=true)) -> h(v)

	should be an onto relationship.

	In other words, a Formula should define a "pure" function.  And we'll let you know if it doesn't.

	### Misc docs:

	- The root filesystem of your execution engine is just another `Input` with the rest, with Location="/".
	Exactly one input with the root location is required at runtime.

	- Formula.SchedulingInfo, since it's *not* included in the 'conjecture', is clearly not expected to have a major impact on your execution correctness.
	This is probably an assumption that's sometimes broken (vms can do more than containers, for example); if so, consider using the
*/
package def

/*
	Formula describes `(inputs, computation) -> (outputs)`.

	Values may be mutated during final validation if missing,
	i.e. the special `Output` that describes stdout and stderr is required
	and will be supplied for you if not already specifically configured.
*/
type Formula struct {
	Inputs         []Input     // total set of inputs.  sorted order.  included in the conjecture.
	SchedulingInfo interface{} // configures what execution framework is used and impl-specific additional parameters to that (minimum node memory, etc).  not considered part of the conjecture.
	Accents        Accents     // additional (executor-independent) constraints and information about a task.  use sparingly.  included in the conjecture.
	Outputs        []Output    // set of expected outputs.  sorted order.  conditionally included in the conjecture (configurable per output).
}

/*
	Input specifies a data source to feed into the beginning of a computation.

	Inputs can be one of many different `Type`s of data source.
	Examples may include "tar", "git", "hadoop", "ipfs", etc.

	Inputs must specify both a `Hash` and a `URL`.
	`Input.Hash` is the content identity descriptor and will always be verified for all types of data source.
	`Input.Hash` is both identifies the data and provides integrity (in other words,
	all repeatr's input types will use a cryptographically strong hash here,
	so given a hash even an untrusted datastore is safe to use).
	Repeatr requires this to be accurate because if the inputs change, output
	consistency is impossible -- so even for plain http downloads, this is enforced.

	`Input.URL` is a secondary content lookup descriptor, like where on
	the filesystem or network information can be fetched from.
	`Input.URL` might contain extra description to answer questions like
	"which git remote url should i fetch from" or
	"which ipfs swarm do i coordinate with".

	The `URL` is *not* included in the conjecture, because repeatr understands
	that your data will be mobile -- that's why we have the `Input.Hash` take the leading role
	and why the `Input.Hash` should be sufficient to identify the information.
	Changes in the `Input.URL` field may make or break whether the data is accessible,
	but should never actually change the content of the data -- it's just supposed to talk about
	transport details; and content itself is still checked by `Input.Hash`.
*/
type Input struct {
	Type     string // implementation name (repeatr-internal).  included in the conjecture.
	Hash     string // identifying hash of input data.  included in the conjecture.
	URL      string // secondary content lookup descriptor.  not considered part of the conjecture.
	Location string // filepath where this input should be mounted in the execution context.  included in the conjecture.
}

/*
	Accents lists executor-independent constraints and information about a task.
	All content is part of the conjecture.

	`OS` and `Arch` constraints may be specified here.  This may be used by the scheduler.
	They are also considered part of the 'conjecture' since it's typically Pretty Hard
	to get things to behave identially across many platforms, so we won't try to
	group together formula that run on different platforms by default.
	(If you want to assert	things are the same across all platforms, great!
	Build a query to gather formulas together to check for that.)

	NOTE: this entire struct is janky; try not to leak too much linux/container specific stuff into it
*/
type Accents struct {
	OS         string            // OS restriction.  use values matching '$GOOS'.  linux presumed if unset.  included in the conjecture.
	Arch       string            // architecture restriction.  use values matching '$GOARCH'.  x86_64 presumed if unset.  included in the conjecture.
	Entrypoint []string          // executable to invoke as the task.  included in the conjecture.
	Env        map[string]string // environment variables.  included in the conjecture.
	Custom     map[string]string // User-defined map; a no-man's land where anything goes.  included in the conjecture.
}

/*
	Output describes where we intend to pick up data after a task completes.

	Outputs can be one of many different `Type`s of data sink.
	Examples may include "tar", "git", "hadoop", "ipfs", etc.

	`Output.Location` states where we should collect information from the
	task execution environment.  Repeatr executors will make sure this
	path exists and is owned&writable by the task before starting.
	After the task completes, repeatr will pick up this data, ship it off
	to storage, and also calculate a checksum of the data so we can see
	whether it matches any prior (or future) runs of this `Formula`.

	Outputs must specify a `URL`; repeatr will ship your data to this address.
	`Output.URL` has similar properties to `Input.URL` (and also similarly,
	is not included in the conjecture, because repeatr understands that
	your data can be mobile).

	The `Output.Hash` field will be filled in with a value computed
	from the data present in `Output.Location` after the task has completed.
	As with `Input.Hash`, the `Output.Hash` in repeatr will always be a
	cryptographically strong hash, which means it precisely describes your
	data, and makes it virtually impossible to accidentally get the same
	`Hash` as other data -- any changes to your output will always result
	in a very different `Hash` value.

	(In a content-addressable data store, repeatr may just lift the data store's
	address to use as `Output.Hash`, which is super efficient for everyone involved.
	For other more legacy-oriented systems, this may be a hash of the
	of the working filesystem right before before export.)

	Whether or not to include an `Output` in the overall `Formula`'s conjecture
	is up to you!  Many things in the world are not deterministic; repeatr
	is here to help you with the ones that should be, and stay out of the way
	for the ones that aren't.  Just set the `Output.Conjecture` boolean.

	Some examples of using `Conjecture` conditionally: if you have a job
	which does a bunch of calculations and should spit out a consistent result,
	but also does a lot of progress logging, gather those in two separate outputs.
	Mark the output of your computation in one output and set that to be
	included in the conjecture so repeatr can help you check your algorithm's
	correctness.  Now, since you may want to keep your logs for later, mark
	those as another output, and since these probably contain timestamps and
	other info that isn't *supposed* to be repeated exactly on another run,
	just set `Conjecture=false` on this one so repeatr knows not to check.
*/
type Output struct {
	Type       string // implementation name (repeatr-internal).  included in the conjecture (iff the whole output is).
	Hash       string // identifying hash of output data.  generated by the output handling implementation during data export when a task is complete.  included in the conjecture (iff the whole output is).
	URL        string // where to ship the output data.  not considered part of the conjecture.
	Location   string // filepath where this output will be yanked from the job when it reaches completion.  included in the conjecture (iff the whole output is).
	Conjecture bool   // whether or not this output is expected to contain the same result, every time, when given the same set of `Input` items.
}

type Job interface {
	// TODO: no idea what goes here, just saying it's a distinct concept from the serializable `Formula` type.

	// Among other things, this should contain progress reporting interfaces, streams that get realtime stdout/stderr, etc.
	// Most of those things will also be accessible as some form of Output after the job is complete, but ActiveJob can provide them live.
}
