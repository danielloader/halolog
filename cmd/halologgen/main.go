// Copyright 2025 Admilson B. F. Cossa
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// halologgen generates a compile-time-checked logging facade from a schema.
//
// The schema names every field a service is allowed to log, with its type.
// The generated package turns each field into a typed method on level-first
// line builders and a context builder — so a misspelled key or a
// wrong-typed value is a COMPILE error, not a corrupt log line. Key
// escaping is computed once at generation time and pinned by a generated
// test, so the escaped form is frozen and reviewed like any other code.
//
// Usage:
//
//	halologgen -schema logging.yaml -out ./applog
//
// Schema:
//
//	package: applog
//	fields:
//	  - name: user_id        # JSON key (required)
//	    type: string         # string|int|int64|float64|bool|error|any (required)
//	    ident: UserID        # Go method name (optional; derived from name)
//
// Author: Admilson B. F. Cossa
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-gen-ecosystem/halolog/cmd/halologgen/internal/gen"
)

func main() {
	schemaPath := flag.String("schema", "", "path to the YAML logging schema (required)")
	outDir := flag.String("out", ".", "directory to write the generated package into")
	flag.Parse()

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "halologgen: -schema is required")
		flag.Usage()
		os.Exit(2)
	}

	if err := gen.Run(*schemaPath, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "halologgen: %v\n", err)
		os.Exit(1)
	}
}
