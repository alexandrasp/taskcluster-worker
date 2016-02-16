// This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go

package winnative

import "github.com/taskcluster/taskcluster-worker/runtime"

type (
	// Config applicable to windows native engine
	Config struct {

		// Whether to use PSExec for executing processes
		UsePsExec bool `json:"usePsExec"`
	}
)

func ConfigSchema() runtime.CompositeSchema {
	schema, err := runtime.NewCompositeSchema(
		"config",
		`
		{
		  "$schema": "http://json-schema.org/draft-04/schema#",
		  "additionalProperties": false,
		  "description": "Config applicable to windows native engine",
		  "properties": {
		    "usePsExec": {
		      "description": "Whether to use PSExec for executing processes",
		      "title": "Use PSExec",
		      "type": "boolean"
		    }
		  },
		  "required": [
		    "usePsExec"
		  ],
		  "title": "Config",
		  "type": "object"
		}
		`,
		true,
		func() interface{} {
			return &Config{}
		},
	)
	if err != nil {
		panic(err)
	}
	return schema
}
