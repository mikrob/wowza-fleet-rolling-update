package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"github.com/coreos/fleet/api"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/log"
	"github.com/coreos/fleet/machine"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/fleet/unit"
)

const (
	defaultSleepTime = 500 * time.Millisecond
)

var (
	machineStates map[string]*machine.MachineState
)

func unitNameMangle(arg string) string {
	return maybeAppendDefaultUnitType(path.Base(arg))
}

func maybeAppendDefaultUnitType(arg string) string {
	if !unit.RecognizedUnitType(arg) {
		arg = unit.DefaultUnitType(arg)
	}
	return arg
}

func checkUnitCreation(arg string, cAPI client.API) (int, error) {
	name := unitNameMangle(arg)

	// First, check if there already exists a Unit by the given name in the Registry
	unit, err := cAPI.Unit(name)
	if err != nil {
		return 1, fmt.Errorf("error retrieving Unit(%s) from Registry: %v", name, err)
	}

	replace := true

	// check if the unit is running
	if unit == nil {
		if replace {
			log.Debugf("Unit(%s) was not found in Registry", name)
		}
		// Create a new unit
		return 0, nil
	}

	// if replace is not set then we warn in case the units differ
	different := false

	// if replace is set then we fail for errors
	if replace {
		if err != nil {
			return 1, err
		} else if different {
			return checkReplaceUnitState(unit)
		} else {
			fmt.Printf("Found same Unit(%s) in Registry, nothing to do", unit.Name)
		}
	} else if different == false {
		log.Debugf("Found same Unit(%s) in Registry, no need to recreate it", name)
	}

	return 1, nil
}

func checkReplaceUnitState(unit *schema.Unit) (int, error) {
	// We replace units only for 'submit', 'load' and
	// 'start' commands.
	allowedReplace := map[string][]job.JobState{
		"submit": []job.JobState{
			job.JobStateInactive,
		},
		"load": []job.JobState{
			job.JobStateInactive,
			job.JobStateLoaded,
		},
		"start": []job.JobState{
			job.JobStateInactive,
			job.JobStateLoaded,
			job.JobStateLaunched,
		},
	}
	currentCommand := "start"
	if allowedJobs, ok := allowedReplace[currentCommand]; ok {
		for _, j := range allowedJobs {
			if job.JobState(unit.DesiredState) == j {
				return 0, nil
			}
		}
		// Report back to caller that we are not allowed to
		// cross unit transition states
		fmt.Printf("Warning: can not replace Unit(%s) in state '%s', use the appropriate command", unit.Name, unit.DesiredState)
	} else {
		// This function should only be called from 'submit',
		// 'load' and 'start' upper paths.
		return 1, fmt.Errorf("error: replacing units is not supported in this context")
	}

	return 1, nil
}

func getUnitFile(file string, cAPI *client.API) (*unit.UnitFile, error) {
	var uf *unit.UnitFile
	name := unitNameMangle(file)

	log.Debugf("Looking for Unit(%s) or its corresponding template", name)

	// Assume that the file references a local unit file on disk and
	// attempt to load it, if it exists
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		uf, err = getUnitFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed getting Unit(%s) from file: %v", file, err)
		}
	} else {
		// Otherwise (if the unit file does not exist), check if the
		// name appears to be an instance of a template unit
		info := unit.NewUnitNameInfo(name)
		if info == nil {
			return nil, fmt.Errorf("error extracting information from unit name %s", name)
		} else if !info.IsInstance() {
			return nil, fmt.Errorf("unable to find Unit(%s) in Registry or on filesystem", name)
		}

		// If it is an instance check for a corresponding template
		// unit in the Registry or disk.
		// If we found a template unit, later we create a
		// near-identical instance unit in the Registry - same
		// unit file as the template, but different name
		uf, err = getUnitFileFromTemplate(cAPI, info, file)
		if err != nil {
			return nil, fmt.Errorf("failed getting Unit(%s) from template: %v", file, err)
		}
	}

	log.Debugf("Found Unit(%s)", name)
	return uf, nil
}

// getUnitFromFile attempts to load a Unit from a given filename
// It returns the Unit or nil, and any error encountered
func getUnitFromFile(file string) (*unit.UnitFile, error) {
	out, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	unitName := path.Base(file)
	log.Debugf("Unit(%s) found in local filesystem", unitName)

	return unit.NewUnitFile(string(out))
}

// getUnitFileFromTemplate attempts to get a Unit from a template unit that
// is either in the registry or on the file system
// It takes two arguments, the template information and the unit file name
// It returns the Unit or nil; and any error encountered
func getUnitFileFromTemplate(cAPI *client.API, uni *unit.UnitNameInfo, fileName string) (*unit.UnitFile, error) {
	var uf *unit.UnitFile

	tmpl, err := (*cAPI).Unit(uni.Template)
	if err != nil {
		return nil, fmt.Errorf("error retrieving template Unit(%s) from Registry: %v", uni.Template, err)
	}

	if tmpl != nil {
		uf = schema.MapSchemaUnitOptionsToUnitFile(tmpl.Options)
		log.Debugf("Template Unit(%s) found in registry", uni.Template)
	} else {
		// Finally, if we could not find a template unit in the Registry,
		// check the local disk for one instead
		filePath := path.Join(path.Dir(fileName), uni.Template)
		if os.Stat(filePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to find template Unit(%s) in Registry or on filesystem", uni.Template)
		}

		uf, err = getUnitFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("unable to load template Unit(%s) from file: %v", uni.Template, err)
		}
	}

	return uf, nil
}

// suToGlobal returns whether or not a schema.Unit refers to a global unit
func suToGlobal(su schema.Unit) bool {
	u := job.Unit{
		Unit: *schema.MapSchemaUnitOptionsToUnitFile(su.Options),
	}
	return u.IsGlobal()
}

func lazyCreateUnits(args []string, cAPI *client.API) error {
	errchan := make(chan error)
	var wg sync.WaitGroup
	for _, arg := range args {
		arg = maybeAppendDefaultUnitType(arg)
		name := unitNameMangle(arg)

		ret, err := checkUnitCreation(arg, *cAPI)
		if err != nil {
			return err
		} else if ret != 0 {
			continue
		}

		// Assume that the name references a local unit file on
		// disk or if it is an instance unit and if so get its
		// corresponding unit
		uf, err := getUnitFile(arg, cAPI)
		if err != nil {
			return err
		}

		_, err = createUnit(name, uf, cAPI)
		if err != nil {
			return err
		}

		wg.Add(1)
		go checkUnitState(name, job.JobStateInactive, 0, os.Stdout, &wg, errchan, cAPI)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	haserr := false
	for msg := range errchan {
		fmt.Printf("Error waiting on unit creation: %v", msg)
		fmt.Println()
		haserr = true
	}

	if haserr {
		return fmt.Errorf("One or more errors creating units")
	}

	return nil
}

func createUnit(name string, uf *unit.UnitFile, cAPI *client.API) (*schema.Unit, error) {
	if uf == nil {
		return nil, fmt.Errorf("nil unit provided")
	}
	u := schema.Unit{
		Name:    name,
		Options: schema.MapUnitFileToSchemaUnitOptions(uf),
	}
	// TODO(jonboulle): this dependency on the API package is awkward, and
	// redundant with the check in api.unitsResource.set, but it is a
	// workaround to implementing the same check in the RegistryClient. It
	// will disappear once RegistryClient is deprecated.
	if err := api.ValidateName(name); err != nil {
		return nil, err
	}
	if err := api.ValidateOptions(u.Options); err != nil {
		return nil, err
	}
	j := &job.Job{Unit: *uf}
	if err := j.ValidateRequirements(); err != nil {
		log.Warningf("Unit %s: %v", name, err)
	}
	err := (*cAPI).CreateUnit(&u)
	if err != nil {
		return nil, fmt.Errorf("failed creating unit %s: %v", name, err)
	}

	log.Debugf("Created Unit(%s) in Registry", name)
	return &u, nil
}

func runStartUnit(args []string, cAPI *client.API) (exit int) {
	if len(args) == 0 {
		fmt.Println("No units given")
		return 0
	}

	if err := lazyCreateUnits(args, cAPI); err != nil {
		fmt.Printf("Error creating units: %v", err)
		fmt.Println()
		return 1
	}

	triggered, err := lazyStartUnits(args, cAPI)
	if err != nil {
		fmt.Printf("Error starting units: %v", err)
		fmt.Println()
		return 1
	}

	var starting []string
	for _, u := range triggered {
		if suToGlobal(*u) {
			fmt.Printf("Triggered global unit %s start", u.Name)
			fmt.Println()
		} else {
			starting = append(starting, u.Name)
		}
	}

	if err := tryWaitForUnitStates(starting, "start", job.JobStateLaunched, getBlockAttempts(), os.Stdout, cAPI); err != nil {
		fmt.Printf("Error waiting for unit states, exit status: %v", err)
		fmt.Println()
		return 1
	}

	if err := tryWaitForSystemdActiveState(starting, getBlockAttempts(), cAPI); err != nil {
		fmt.Printf("Error waiting for systemd unit states, err: %v", err)
		fmt.Println()
		return 1
	}

	return 0
}

// tryWaitForSystemdActiveState tries to wait for systemd units to reach an
// active state, making use of cAPI. It takes one or more units as input, and
// ensures that every unit in the []units must be in the active state.
// If yes, return nil. Otherwise return error.
func tryWaitForSystemdActiveState(units []string, maxAttempts int, cAPI *client.API) (err error) {
	if maxAttempts <= -1 {
		for _, name := range units {
			fmt.Printf("Triggered unit %s start", name)
			fmt.Println()
		}
		return nil
	}

	errchan := waitForSystemdActiveState(units, maxAttempts, cAPI)
	for err := range errchan {
		fmt.Printf("Error waiting for units: %v", err)
		fmt.Println()
		return err
	}

	return nil
}

func checkSystemdActiveState(apiStates []*schema.UnitState, name string, maxAttempts int, wg *sync.WaitGroup, errchan chan error, cAPI *client.API) {
	defer wg.Done()

	// "isInf == true" means "blocking forever until it succeeded".
	// In that case, maxAttempts is set to an arbitrary large integer number.
	var isInf bool
	if maxAttempts < 1 {
		isInf = true
		maxAttempts = math.MaxInt32
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := assertFetchSystemdActiveState(apiStates, name, cAPI); err == nil {
			return
		} else {
			errchan <- err
		}

		if !isInf {
			errchan <- fmt.Errorf("timed out waiting for unit %s to report active state", name)
		}
	}
}

func assertFetchSystemdActiveState(apiStates []*schema.UnitState, name string, cAPI *client.API) error {
	if err := assertSystemdActiveState(apiStates, name); err == nil {
		return nil
	}

	// If the assertion failed, we again need to get unit states via cAPI,
	// to retry the assertion repeatedly.
	//
	// NOTE: Ideally we should be able to fetch the state only for a single
	// unit. However, we cannot do that for now, because cAPI.UnitState()
	// is not available. In the future we need to implement cAPI.UnitState()
	// and all dependendent parts all over the tree in fleet, (schema,
	// etcdRegistry, rpcRegistry, etc) to replace UnitStates() in this place
	// with the new method UnitState(). In practice, calling UnitStates() here
	// is not as badly inefficient as it looks, because it will be anyway
	// rarely called only when the assertion failed. - dpark 20160907

	time.Sleep(defaultSleepTime)

	var errU error
	apiStates, errU = (*cAPI).UnitStates()
	if errU != nil {
		return fmt.Errorf("Error retrieving list of units: %v", errU)
	}
	return nil
}

// assertSystemdActiveState determines if a given systemd unit is actually
// in the active state, making use of cAPI
func assertSystemdActiveState(apiStates []*schema.UnitState, unitName string) error {
	uState, err := getSingleUnitState(apiStates, unitName)
	if err != nil {
		return err
	}

	// Get systemd state and check the state is active & loaded.
	if uState.ActiveState != "active" || uState.LoadState != "loaded" {
		return fmt.Errorf("Failed to find an active unit %s", unitName)
	}

	return nil
}

// getSingleUnitState returns a single uState of type suState, which consists
// of necessary systemd states, only for a given unit name.
func getSingleUnitState(apiStates []*schema.UnitState, unitName string) (suState, error) {
	for _, us := range apiStates {
		if us.Name == unitName {
			return suState{
				us.SystemdLoadState,
				us.SystemdActiveState,
				us.SystemdSubState,
			}, nil
		}
	}
	return suState{}, fmt.Errorf("unit %s not found", unitName)
}

type suState struct {
	LoadState   string
	ActiveState string
	SubState    string
}

// waitForSystemdActiveState tries to assert that the given unit becomes
// active, making use of multiple goroutines that check unit states.
func waitForSystemdActiveState(units []string, maxAttempts int, cAPI *client.API) (errch chan error) {
	apiStates, err := (*cAPI).UnitStates()
	if err != nil {
		errch <- fmt.Errorf("Error retrieving list of units: %v", err)
		return
	}

	errchan := make(chan error)
	var wg sync.WaitGroup
	for _, name := range units {
		wg.Add(1)
		go checkSystemdActiveState(apiStates, name, maxAttempts, &wg, errchan, cAPI)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	return errchan
}

// getBlockAttempts gets the correct value of how many attempts to try
// before giving up on an operation.
// It returns a negative value which means do not block, if zero is
// returned then it means try forever, and if a positive value is
// returned then try up to that value
func getBlockAttempts() int {
	return 0
}

// tryWaitForUnitStates tries to wait for units to reach the desired state.
// It takes 5 arguments, the units to wait for, the desired state, the
// desired JobState, how many attempts before timing out and a writer
// interface.
// tryWaitForUnitStates polls each of the indicated units until they
// reach the desired state. If maxAttempts is negative, then it will not
// wait, it will assume that all units reached their desired state.
// If maxAttempts is zero tryWaitForUnitStates will retry forever, and
// if it is greater than zero, it will retry up to the indicated value.
// It returns nil on success or error on failure.
func tryWaitForUnitStates(units []string, state string, js job.JobState, maxAttempts int, out io.Writer, cAPI *client.API) error {
	// We do not wait just assume we reached the desired state
	if maxAttempts <= -1 {
		for _, name := range units {
			fmt.Printf("Triggered unit %s %s", name, state)
			fmt.Println()
		}
		return nil
	}

	errchan := waitForUnitStates(units, js, maxAttempts, out, cAPI)
	for err := range errchan {
		fmt.Printf("Error waiting for units: %v", err)
		fmt.Println()
		return err
	}

	return nil
}

func checkUnitState(name string, js job.JobState, maxAttempts int, out io.Writer, wg *sync.WaitGroup, errchan chan error, cAPI *client.API) {
	defer wg.Done()

	sleep := defaultSleepTime

	if maxAttempts < 1 {
		for {
			if assertUnitState(name, js, out, cAPI) {
				return
			}
			time.Sleep(sleep)
		}
	} else {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if assertUnitState(name, js, out, cAPI) {
				return
			}
			time.Sleep(sleep)
		}
		errchan <- fmt.Errorf("timed out waiting for unit %s to report state %s", name, js)
	}
}

func assertUnitState(name string, js job.JobState, out io.Writer, cAPI *client.API) (ret bool) {
	var state string

	u, err := (*cAPI).Unit(name)
	if err != nil {
		log.Warningf("Error retrieving Unit(%s) from Registry: %v", name, err)
		return
	}
	if u == nil {
		log.Warningf("Unit %s not found", name)
		return
	}

	// If this is a global unit, CurrentState will never be set. Instead, wait for DesiredState.
	if suToGlobal(*u) {
		state = u.DesiredState
	} else {
		state = u.CurrentState
	}

	if job.JobState(state) != js {
		log.Debugf("Waiting for Unit(%s) state(%s) to be %s", name, job.JobState(state), js)
		return
	}

	ret = true
	msg := fmt.Sprintf("Unit %s %s", name, u.CurrentState)

	if u.MachineID != "" {
		ms := cachedMachineState(u.MachineID, cAPI)
		if ms != nil {
			msg = fmt.Sprintf("%s on %s", msg, machineFullLegend(*ms, false))
		}
	}

	fmt.Fprintln(out, msg)
	return
}

func machineIDLegend(ms machine.MachineState, full bool) string {
	legend := ms.ID
	if !full {
		legend = fmt.Sprintf("%s...", ms.ShortID())
	}
	return legend
}

func machineFullLegend(ms machine.MachineState, full bool) string {
	legend := machineIDLegend(ms, full)
	if len(ms.PublicIP) > 0 {
		legend = fmt.Sprintf("%s/%s", legend, ms.PublicIP)
	}
	return legend
}

// cachedMachineState makes a best-effort to retrieve the MachineState of the given machine ID.
// It memoizes MachineState information for the life of a fleetctl invocation.
// Any error encountered retrieving the list of machines is ignored.
func cachedMachineState(machID string, cAPI *client.API) (ms *machine.MachineState) {
	if machineStates == nil {
		machineStates = make(map[string]*machine.MachineState)
		ms, err := (*cAPI).Machines()
		if err != nil {
			return nil
		}
		for i, m := range ms {
			machineStates[m.ID] = &ms[i]
		}
	}
	return machineStates[machID]
}

// waitForUnitStates polls each of the indicated units until each of their
// states is equal to that which the caller indicates, or until the
// polling operation times out. waitForUnitStates will retry forever, or
// up to maxAttempts times before timing out if maxAttempts is greater
// than zero. Returned is an error channel used to communicate when
// timeouts occur. The returned error channel will be closed after all
// polling operation is complete.
func waitForUnitStates(units []string, js job.JobState, maxAttempts int, out io.Writer, cAPI *client.API) chan error {
	errchan := make(chan error)
	var wg sync.WaitGroup
	for _, name := range units {
		wg.Add(1)
		go checkUnitState(name, js, maxAttempts, out, &wg, errchan, cAPI)
	}

	go func() {
		wg.Wait()
		close(errchan)
	}()

	return errchan
}

func lazyStartUnits(args []string, cAPI *client.API) ([]*schema.Unit, error) {
	units := make([]string, 0, len(args))
	for _, j := range args {
		units = append(units, unitNameMangle(j))
	}
	return setTargetStateOfUnits(units, job.JobStateLaunched, cAPI)
}

// setTargetStateOfUnits ensures that the target state for the given Units is set
// to the given state in the Registry.
// On success, a slice of the Units for which a state change was made is returned.
// Any error encountered is immediately returned (i.e. this is not a transaction).
func setTargetStateOfUnits(units []string, state job.JobState, cAPI *client.API) ([]*schema.Unit, error) {
	triggered := make([]*schema.Unit, 0)
	for _, name := range units {
		u, err := (*cAPI).Unit(name)
		if err != nil {
			return nil, fmt.Errorf("error retrieving unit %s from registry: %v", name, err)
		} else if u == nil {
			return nil, fmt.Errorf("unable to find unit %s", name)
		} else if job.JobState(u.DesiredState) == state {
			log.Debugf("Unit(%s) already %s, skipping.", u.Name, u.DesiredState)
			continue
		}

		log.Debugf("Setting Unit(%s) target state to %s", u.Name, state)
		if err := (*cAPI).SetUnitTargetState(u.Name, string(state)); err != nil {
			return nil, err
		}
		triggered = append(triggered, u)
	}

	return triggered, nil
}
