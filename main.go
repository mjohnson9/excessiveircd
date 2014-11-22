// Copyright (c) 2014 Michael Johnson. All rights reserved.
//
// Use of this source code is governed by the BSD license that can be found in
// the LICENSE file.

package main

import (
	"flag"

	"github.com/nightexcessive/excessiveircd/server"
)

func main() {
	flag.Parse()

	server.Start()
}
