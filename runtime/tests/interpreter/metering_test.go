/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretMetering(t *testing.T) {

	checkerImported, err := ParseAndCheck(t, `
      pub fun a() {
          true
          true
      }
    `)
	require.NoError(t, err)

	checkerImporting, err := ParseAndCheckWithOptions(t,
		`
          import a from "imported"

          fun b() {
              true
              true
              a()
              true
              true
          }

          fun c() {
              true
              true
              b()
              true
              true
          }
        `,
		ParseAndCheckOptions{
			ImportResolver: func(location ast.Location) (program *ast.Program, e error) {
				return checkerImported.Program, nil
			},
		},
	)
	require.NoError(t, err)

	type occurrence struct {
		interpreterID int
		line          int
	}

	var occurrences []occurrence
	var nextInterpreterID int
	interpreterIDs := map[*interpreter.Interpreter]int{}

	inter, err := interpreter.NewInterpreter(
		checkerImporting,
		interpreter.WithOnStatementHandler(
			func(statement *interpreter.Statement) {
				inter := statement.Interpreter

				id, ok := interpreterIDs[inter]
				if !ok {
					id = nextInterpreterID
					nextInterpreterID++
					interpreterIDs[inter] = id
				}

				occurrences = append(occurrences, occurrence{
					interpreterID: id,
					line:          statement.Line,
				})
			},
		),
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	_, err = inter.Invoke("c")
	require.NoError(t, err)

	assert.Equal(t,
		[]occurrence{
			{0, 13},
			{0, 14},
			{0, 15},
			{0, 5},
			{0, 6},
			{0, 7},
			{1, 3},
			{1, 4},
			{0, 8},
			{0, 9},
			{0, 16},
			{0, 17},
		},
		occurrences,
	)
}
