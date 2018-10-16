package null

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func resource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Read:   resourceRead,
		Delete: resourceDelete,

		Schema: map[string]*schema.Schema{
			"triggers": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"external_trigger": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"command": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"interpreter": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"working_dir": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"environment": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
						"result": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func runCommand(data *schema.ResourceData) error {
	command := data.Get("command").(string)
	if command == "" {
		return fmt.Errorf("local-exec provisioner command must be a non-empty string")
	}

	// Execute the command with env
	environment := data.Get("environment").(map[string]interface{})

	var env []string
	for k := range environment {
		entry := fmt.Sprintf("%s=%s", k, environment[k].(string))
		env = append(env, entry)
	}

	// Execute the command using a shell
	interpreter := data.Get("interpreter").([]interface{})

	var cmdargs []string
	if len(interpreter) > 0 {
		for _, i := range interpreter {
			if arg, ok := i.(string); ok {
				cmdargs = append(cmdargs, arg)
			}
		}
	} else {
		if runtime.GOOS == "windows" {
			cmdargs = []string{"cmd", "/C"}
		} else {
			cmdargs = []string{"/bin/sh", "-c"}
		}
	}
	cmdargs = append(cmdargs, command)

	workingdir := data.Get("working_dir").(string)

	var cmdEnv []string
	cmdEnv = os.Environ()
	cmdEnv = append(cmdEnv, env...)

	// Setup the command
	cmd := exec.Command(cmdargs[0], cmdargs[1:]...)
	// Dir specifies the working directory of the command.
	// If Dir is the empty string (this is default), runs the command
	// in the calling process's current directory.
	cmd.Dir = workingdir
	// Env specifies the environment of the command.
	// By default will use the calling process's environment
	cmd.Env = cmdEnv

	// Start the command
	result, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Stderr != nil && len(exitErr.Stderr) > 0 {
				return fmt.Errorf("failed to execute %q: %s", command, string(exitErr.Stderr))
			}
			return fmt.Errorf("command %q failed with no error message", command)
		} else {
			return fmt.Errorf("failed to execute %q: %s", command, err)
		}
	}
	data.Set("result", result)

	return nil
}

func runExternalTriggers(d *schema.ResourceData) error {
	d2, ok := d.GetOk("external_trigger")
	if !ok {
		return nil
	}

	for _, triggerRaw := range d2.([]interface{}) {
		trigger := triggerRaw.(*schema.ResourceData)
		err := runCommand(trigger)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	err := runExternalTriggers(d)
	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%d", rand.Int()))

	return nil
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	return runExternalTriggers(d)
}

func resourceDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
