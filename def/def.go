package def

type JobRecord struct {
	ID             string       // ID is a guid, and can be considered a primary key.  (We need some way to refer to different executions with the same inputs, just so we can disambiguate tries.)
	Inputs         []Input      // The total set of inputs.  This should be maintained in sorted order.
	SchedulingInfo interface{}  // Scheduling info configures what execution framework is used and some additional parameters to that (minimum node memory, etc).  None of these values will be considered part of the 'conjecture' inputs.
	Accents        Accents      // Accents allow placing additional constraints and information into a job.  Use sparingly.
	Outputs        []OutputSpec // The set of expected outputs.  (Some of these may be added, if missing, i.e. the special `OutputSpec` that describes stdout and stderr.)
}

type Input struct {
	Type     string // `Input.Type` is the repeatr-internal name of what kind of data source this is.  It specificies a handling algorithm.  Examples may include "tar", "git", "hadoop", "ipfs", etc.
	Hash     string // `Input.Hash` is the content identity descriptor.  In bad systems, this is merely a verifier.  In good systems, it's actually the lookup key itself.  Repeatr requires this to be accurate because if the inputs change, output consistency is impossible -- so even for plain http downloads, this is enforced.
	URL      string // `Input.URL` is a secondary content lookup descriptor.  The `Input.Hash` should be sufficient to identify the information; but this field may contain extra description like "which ipfs swarm do i coordinate with".  (Changes in this field may make or break whether the data is accessible, but should never actually change the content of the data -- this is just transport details; content itself is still checked by `Input.Hash`.)
	Location string // `Input.Location` is the filepath where this input should be provided to the job.
}

// TODO: this entire struct is janky; try not to leak too much linux/container specific stuff into it
type Accents struct {
	OS         string            // Specify an OS limitation.  This may be used by the scheduler, and is also considered part of the 'conjecture' (if you want to assert things are the same across all platforms, build a query to check for that).
	Arch       string            // Specify an architecture limitation.  Similar to `Accents.OS`.
	Entrypoint []string          // Executable to invoke as the job.
	Env        map[string]string // Environment variables
}

type OutputSpec struct {
	Type       string // `OutputSpec.Type` is the repeatr-internal name of what kind of data sink to shovel this output into.  It specifies a handling algorithm.
	Location   string // `OutputSpec.Location` is the filepath where this output will be yanked from the job when it reaches completion.
	URL        string // `OutputSpec.URL` is a secondary content placement descriptor.  Like Input.URL, it should not be considered to identify the information (a hash is needed for that; an `Output` struct will be generated by the commission of any outputs, and that will contain the suitable `Output.Hash`); but this field may contain extra description like "which ipfs swarm do i coordinate with".
	Conjecture bool   // If an `Output` is part of a job's 'conjecture', it is expected to produce the same result, every time, when given the same set of `Input` items.  This can be used to verify correct repeatability of a process.  An example of using (and not using) this: the set of which tests pass and fail should be consistent for a given input git repo, so output lists of pass/fail test names in an output that is conjecture=true; meanwhile, the logs from the tests probably contain timestamps and so aren't part of precise repeatability, so while we want to keep those for reading, they belong in another output that's conjecture=false.
}

type Output struct {
	OutputSpec
	Hash string // `Output.Hash` is generated by the output handling implementation.  (In a content-addressible filesystem, we can just borrow their concept.  For other more legacy-oriented systems, this may be a hash of the contents produced from the job's working filesystem before export.)
}

type ActiveJob interface {
	// TODO: no idea what goes here, just saying it's a distinct concept from the serializable `JobRecord` type.

	// Among other things, this should contain progress reporting interfaces, streams that get realtime stdout/stderr, etc.
	// Most of those things will also be accessible as some form of Output after the job is complete, but ActiveJob can provide them live.
}

/*
	Critial focus:

	Given a JobRecord j, and the []Output v, and some hash h:

	h(j.Inputs||j.Accents||filter(j.Outputs, where Conjecture=true)) -> h(v)

	should be an onto relationship.

	In other words, a JobRecord should define a "pure" function.  And we'll let you know if it doesn't.



	Misc docs:

	- The root filesystem of your execution engine is just another `Input` with the rest, with Location="/".
	Exactly one input with the root location is required at runtime.

	- JobRecord.SchedulingInfo, since it's *not* included in the 'conjecture', is clearly not expected to have a major impact on your execution correctness.
	This is probably an assumption that's sometimes broken (vms can do more than containers, for example); if so, consider using the
*/
