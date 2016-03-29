// This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go

package artifacts

import (
	"github.com/taskcluster/taskcluster-client-go/tcclient"
	"github.com/taskcluster/taskcluster-worker/runtime"
)

type (
	// Artifacts to be published
	payload []struct {

		// Date when artifact should expire must be in the future
		Expires tcclient.Time `json:"expires,omitempty"`

		// This will be the leading path to directories and the full name for files that are uploaded to s3
		//
		// Syntax:     ^[^/].*[^/]$
		Name string `json:"name"`

		// Filesystem path of artifact
		//
		// Syntax:     ^.*[^/]$
		Path string `json:"path"`

		// Artifacts can be either an individual `file` or a `directory` containing potentially multiple files with recursively included subdirectories
		//
		// Possible values:
		//   * "file"
		//   * "directory"
		Type string `json:"type"`
	}
)

var payloadSchema = func() runtime.CompositeSchema {
	schema, err := runtime.NewCompositeSchema(
		"artifacts",
		`
		{
		  "$schema": "http://json-schema.org/draft-04/schema#",
		  "description": "Artifacts to be published",
		  "items": {
		    "properties": {
		      "expires": {
		        "description": "Date when artifact should expire must be in the future",
		        "format": "date-time",
		        "title": "Expiry Tate and Time",
		        "type": "string"
		      },
		      "name": {
		        "description": "This will be the leading path to directories and the full name for files that are uploaded to s3",
		        "pattern": "^[^/].*[^/]$",
		        "title": "Artifact Name",
		        "type": "string"
		      },
		      "path": {
		        "description": "Filesystem path of artifact",
		        "pattern": "^.*[^/]$",
		        "title": "Artifact Path",
		        "type": "string"
		      },
		      "type": {
		        "description": "Artifacts can be either an individual `+"`"+`file`+"`"+` or a `+"`"+`directory`+"`"+` containing potentially multiple files with recursively included subdirectories",
		        "enum": [
		          "file",
		          "directory"
		        ],
		        "title": "Upload type",
		        "type": "string"
		      }
		    },
		    "required": [
		      "type",
		      "path",
		      "name"
		    ],
		    "type": "object"
		  },
		  "title": "payload",
		  "type": "array"
		}
		`,
		false,
		func() interface{} {
			return &payload{}
		},
	)
	if err != nil {
		panic(err)
	}
	return schema
}()
