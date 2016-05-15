// This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go

package qemuengine

import "github.com/taskcluster/taskcluster-worker/runtime"

type (
	engineConfig struct {

		// Path to folder to be used for image storage and cache.
		// Please ensure this has lots of space.
		ImageFolder string `json:"imageFolder"`

		// Maximum number of current VMs the engine is allowed to run.
		//
		// Mininum:    1
		// Maximum:    100
		MaxConcurrency int `json:"maxConcurrency"`

		// Path to folder to be used for internal unix-domain sockets.
		// Ideally, this shouldn't be readable by anyone else.
		SocketFolder string `json:"socketFolder,omitempty"`
	}
)

var engineConfigSchema = func() runtime.CompositeSchema {
	schema, err := runtime.NewCompositeSchema(
		"qemu",
		`
		{
		  "$schema": "http://json-schema.org/draft-04/schema#",
		  "additionalProperties": false,
		  "properties": {
		    "imageFolder": {
		      "description": "Path to folder to be used for image storage and cache.\nPlease ensure this has lots of space.\n",
		      "type": "string"
		    },
		    "maxConcurrency": {
		      "description": "Maximum number of current VMs the engine is allowed to run.\n",
		      "maximum": 100,
		      "minimum": 1,
		      "type": "integer"
		    },
		    "socketFolder": {
		      "description": "Path to folder to be used for internal unix-domain sockets.\nIdeally, this shouldn't be readable by anyone else.\n",
		      "type": "string"
		    }
		  },
		  "required": [
		    "maxConcurrency",
		    "imageFolder"
		  ],
		  "title": "Engine Config",
		  "type": "object"
		}
		`,
		true,
		func() interface{} {
			return &engineConfig{}
		},
	)
	if err != nil {
		panic(err)
	}
	return schema
}()