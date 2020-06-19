// Copyright 2019 The go-dsplinz Authors
// This file is part of the go-dsplinz library.
//
// The go-dsplinz library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-dsplinz library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-dsplinz library. If not, see <http://www.gnu.org/licenses/>.

// +build gofuzz

package runtime

// Fuzz is the basic entry point for the go-fuzz tool
//
// This returns 1 for valid parsable/runable code, 0
// for invalid opcode.
func Fuzz(input []byte) int {
	_, _, err := Execute(input, input, &Config{
		GasLimit: 3000000,
	})

	// invalid opcode
	if err != nil && len(err.Error()) > 6 && string(err.Error()[:7]) == "invalid" {
		return 0
	}

	return 1
}
