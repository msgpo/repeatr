package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor"
	"polydawn.net/repeatr/scheduler"
)

func LoadFormulaFromFile(path string) def.Formula {
	filename, _ := filepath.Abs(path)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(Error.Wrap(fmt.Errorf("Could not read formula file %q: %s", filename, err)))
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	var raw interface{}
	if err := dec.Decode(&raw); err != nil {
		panic(Error.New("Could not parse formula file %q: %s", filename, err))
	}
	raw = stringifyMapKeys(raw)
	formula := def.Formula{}
	if err := formula.Unmarshal(raw); err != nil {
		panic(Error.New("Could not parse formula file %q: %s", filename, err))
	}

	return formula
}

func stringifyMapKeys(value interface{}) interface{} {
	switch value := value.(type) {
	case map[interface{}]interface{}:
		next := make(map[string]interface{}, len(value))
		for k, v := range value {
			next[k.(string)] = stringifyMapKeys(v)
		}
		return next
	case []interface{}:
		for i := 0; i < len(value); i++ {
			value[i] = stringifyMapKeys(value[i])
		}
		return value
	default:
		return value
	}
}

func RunFormula(s scheduler.Scheduler, e executor.Executor, formula def.Formula, journal io.Writer) def.JobResult {
	jobLoggerFactory := func(_ def.JobID) io.Writer {
		// All job progress reporting, still copy to our shared journal stream.
		// This func might now be outdated; but we haven't decided what any of this
		//  should look like if take a lurch toward supporting cluster farming.
		//  (It might make sense to have a structural comms layer?  Or, maybe plain
		//  byte streams are best for sanity conservation.  Either way: not today.)
		return journal
	}

	s.Configure(e, 1, jobLoggerFactory) // queue concept a bit misplaced here
	s.Start()

	// Set up a logger.
	log := log15.New()
	log.SetHandler(log15.StreamHandler(journal, log15.TerminalFormat()))

	id, jobChan := s.Schedule(formula)
	log = log.New(log15.Ctx{"JobID": id})

	log.Info("Job queued")
	job := <-jobChan
	// TODO need better lifecycle events here.  "starting" here means we might still be in provisioning stage.
	log.Info("Job starting")

	// Stream job output to terminal in real time
	_, err := io.Copy(journal, job.OutputReader())
	if err != nil {
		log.Error("Error reading job stream", "error", err)
		panic(err)
	}

	result := job.Wait()
	if result.Error != nil {
		log.Error("Job execution errored", "error", result.Error.Message())
	} else {
		log.Info("Job finished", log15.Ctx{
			"exit":    result.ExitCode,
			"outputs": result.Outputs,
		})
	}
	return result
}
