// Copyright 2019 The go-dsplinz Authors
// This file is part of go-dsplinz.
//
// go-dsplinz is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-dsplinz is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-dsplinz. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dsplinz2019/dsplinz/cmd/utils"
	swarm "github.com/dsplinz2019/dsplinz/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

func list(ctx *cli.Context) {
	args := ctx.Args()

	if len(args) < 1 {
		utils.Fatalf("Please supply a manifest reference as the first argument")
	} else if len(args) > 2 {
		utils.Fatalf("Too many arguments - usage 'swarm ls manifest [prefix]'")
	}
	manifest := args[0]

	var prefix string
	if len(args) == 2 {
		prefix = args[1]
	}

	bzzapi := strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
	client := swarm.NewClient(bzzapi)
	list, err := client.List(manifest, prefix)
	if err != nil {
		utils.Fatalf("Failed to generate file and directory list: %s", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "HASH\tCONTENT TYPE\tPATH")
	for _, prefix := range list.CommonPrefixes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", "", "DIR", prefix)
	}
	for _, entry := range list.Entries {
		fmt.Fprintf(w, "%s\t%s\t%s\n", entry.Hash, entry.ContentType, entry.Path)
	}
}
